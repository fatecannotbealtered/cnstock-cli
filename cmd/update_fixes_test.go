package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatecannotbealtered/cnstock-cli/internal/api"
	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
)

// newUpdateReleaseServerBundleStatus serves a healthy release (archive + checksums
// download fine) but returns bundleStatus for the signature-bundle download, so a
// test can exercise a transient bundle-fetch failure independently of signature
// verification.
func newUpdateReleaseServerBundleStatus(t *testing.T, tag string, bundleStatus int) *httptest.Server {
	t.Helper()
	var archive bytes.Buffer
	gzw := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gzw)
	payload := []byte("new-binary-bytes")
	_ = tw.WriteHeader(&tar.Header{Name: "cnstock-cli", Mode: 0o755, Size: int64(len(payload)), Typeflag: tar.TypeReg})
	_, _ = tw.Write(payload)
	_ = tw.Close()
	_ = gzw.Close()
	archiveBytes := archive.Bytes()

	assetName := fmt.Sprintf("cnstock-cli-%s-linux-amd64.tar.gz", strings.TrimPrefix(tag, "v"))
	sum := sha256.Sum256(archiveBytes)
	checksums := hex.EncodeToString(sum[:]) + "  " + assetName + "\n"

	mux := http.NewServeMux()
	var base string
	releaseJSON := func() string {
		rel := map[string]any{
			"tag_name": tag,
			"html_url": "https://example.com/releases/" + tag,
			"assets": []map[string]any{
				{"name": assetName, "browser_download_url": base + "/dl/archive"},
				{"name": "checksums.txt", "browser_download_url": base + "/dl/checksums"},
				{"name": "checksums.txt.sigstore.json", "browser_download_url": base + "/dl/bundle"},
			},
		}
		b, _ := json.Marshal(rel)
		return string(b)
	}
	mux.HandleFunc("/repos/", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, releaseJSON()) })
	mux.HandleFunc("/latest", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, releaseJSON()) })
	mux.HandleFunc("/dl/archive", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archiveBytes) })
	mux.HandleFunc("/dl/checksums", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, checksums) })
	mux.HandleFunc("/dl/bundle", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(bundleStatus) })

	srv := httptest.NewServer(mux)
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

// Fix #2: discover-stage HTTP failures must be classified by status onto the
// taxonomy, not collapsed into E_NETWORK. fetchBinaryRelease routes non-2xx
// through api.ErrorForStatus.
func TestFetchBinaryRelease_StatusClassification(t *testing.T) {
	tests := []struct {
		name   string
		status int
		wantAs func(error) bool
	}{
		{"not_found", http.StatusNotFound, func(e error) bool { var t *api.NotFoundError; return errors.As(e, &t) }},
		{"timeout", http.StatusRequestTimeout, func(e error) bool { var t *api.TimeoutError; return errors.As(e, &t) }},
		{"rate_limited", http.StatusTooManyRequests, func(e error) bool { var t *api.RateLimitError; return errors.As(e, &t) }},
		{"server", http.StatusInternalServerError, func(e error) bool { var t *api.ServerError; return errors.As(e, &t) }},
	}
	origAPI, origClient := updateBinaryGitHubAPI, updateBinaryHTTPClient
	t.Cleanup(func() { updateBinaryGitHubAPI, updateBinaryHTTPClient = origAPI, origClient })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			updateBinaryGitHubAPI = srv.URL
			updateBinaryHTTPClient = srv.Client()

			_, err := fetchBinaryRelease(context.Background(), "")
			if err == nil {
				t.Fatalf("status %d: expected error", tt.status)
			}
			if !tt.wantAs(err) {
				t.Fatalf("status %d mapped to wrong error type: %v", tt.status, err)
			}
		})
	}
}

// Fix #2 end-to-end: a discover 404 surfaces as non-retryable E_NOT_FOUND (exit 3)
// through runUpdate, not a retryable E_NETWORK.
func TestUpdate_DiscoverNotFoundIsNonRetryable(t *testing.T) {
	// Release-available probe must succeed (so we proceed to performBinaryUpdate),
	// then the binary-release discover returns 404.
	relSrv := newUpdateReleaseServer(t, "v9.9.9")
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(notFound.Close)

	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(relSrv, okApply, okSkillSync)
		// Point only the binary-release discover at the 404 host; the update-available
		// probe still uses the healthy release server endpoint.
		updateBinaryGitHubAPI = notFound.URL
	})
	if exit != ExitNotFound {
		t.Fatalf("exit = %d, want 3 (E_NOT_FOUND); env: %v", exit, env)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "E_NOT_FOUND" {
		t.Errorf("code = %v, want E_NOT_FOUND", errObj["code"])
	}
	if errObj["retryable"] != false {
		t.Errorf("retryable = %v, want false", errObj["retryable"])
	}
	details, _ := errObj["details"].(map[string]any)
	if details["stage"] != updateStageDiscover {
		t.Errorf("stage = %v, want discover", details["stage"])
	}
	if details["binary_replaced"] != false {
		t.Errorf("binary_replaced = %v, want false", details["binary_replaced"])
	}
}

// Fix #2 (regression): the latest-release probe — the FIRST network call of every
// update and the notice-refresh path — must classify a 404 by status too. A real
// 404 here means the release/repo is gone, which is non-retryable E_NOT_FOUND
// (exit 3), not a retryable E_SERVER. This drives the 404 through fetchLatestRelease
// itself (the prior test stubbed this probe to succeed and only 404'd the binary
// discover, masking this path).
func TestUpdate_LatestReleaseProbeNotFoundIsNonRetryable(t *testing.T) {
	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(notFound.Close)

	env, exit := captureUpdateRun(t, func() {
		// Stub the downstream seams so that, if the probe were (wrongly) to
		// succeed, the run would proceed rather than fail elsewhere — making this
		// assertion specifically about the probe's status classification.
		stubUpdateSeams(notFound, okApply, okSkillSync)
		// Point the latest-release probe itself at the 404 host.
		_ = os.Setenv("CNS_UPDATE_ENDPOINT", notFound.URL+"/releases/latest")
	})
	if exit != ExitNotFound {
		t.Fatalf("exit = %d, want 3 (E_NOT_FOUND); env: %v", exit, env)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "E_NOT_FOUND" {
		t.Errorf("code = %v, want E_NOT_FOUND", errObj["code"])
	}
	if errObj["retryable"] != false {
		t.Errorf("retryable = %v, want false", errObj["retryable"])
	}
}

// Fix #1: a failure to FETCH the signature bundle is a transient server failure
// (retryable), NOT a non-retryable E_INTEGRITY supply-chain failure. Only a
// signature that fails to verify is E_INTEGRITY.
func TestUpdate_SignatureBundleDownloadFailureIsRetryable(t *testing.T) {
	srv := newUpdateReleaseServerBundleStatus(t, "v9.9.9", http.StatusServiceUnavailable)
	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, okApply, okSkillSync)
		// The verify seam must never be reached because the bundle never downloads.
		updateVerifySignature = func(_, _, _ string) error {
			t.Fatal("signature verify must not run when the bundle download failed")
			return nil
		}
	})
	if exit == ExitGeneric {
		t.Fatalf("bundle download failure must NOT be E_INTEGRITY (exit 1); env: %v", env)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] == "E_INTEGRITY" {
		t.Fatalf("bundle download failure misclassified as E_INTEGRITY; env: %v", env)
	}
	if errObj["retryable"] != true {
		t.Errorf("retryable = %v, want true (transient bundle fetch); env: %v", errObj["retryable"], env)
	}
	details, _ := errObj["details"].(map[string]any)
	if details["stage"] != updateStageDownload {
		t.Errorf("stage = %v, want download (bundle fetch), not verify_signature", details["stage"])
	}
	if details["binary_replaced"] != false {
		t.Errorf("binary_replaced = %v, want false", details["binary_replaced"])
	}
}

// Fix #1: a SIGINT/ctx cancellation during the verify stage emits the terminal
// E_INTERRUPTED envelope (exit 130) stating the old version is still installed,
// never an E_INTEGRITY. The verify seam cancels the run's parent context to
// simulate a Ctrl-C arriving mid-verify; the cancellation must dominate the
// integrity classification.
func TestUpdate_InterruptDuringVerifyIsInterrupted(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	origCtx := updateCmd.Context()
	updateCmd.SetContext(ctx)
	t.Cleanup(func() { updateCmd.SetContext(origCtx) })

	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, okApply, okSkillSync)
		updateVerifySignature = func(_, _, _ string) error {
			cancel() // Ctrl-C arrives while verifying.
			return errors.New("certificate identity mismatch")
		}
	})
	if exit != ExitInterrupted {
		t.Fatalf("exit = %d, want 130 (E_INTERRUPTED); env: %v", exit, env)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "E_INTERRUPTED" {
		t.Errorf("code = %v, want E_INTERRUPTED", errObj["code"])
	}
	details, _ := errObj["details"].(map[string]any)
	if details["binary_replaced"] != false {
		t.Errorf("binary_replaced = %v, want false (still on old version)", details["binary_replaced"])
	}
	if details["current_version"] != version {
		t.Errorf("current_version = %v, want %s (unchanged)", details["current_version"], version)
	}
}

// Fix #3: detectInstallMethod probes the real install layout instead of returning
// a hardcoded value.
func TestDetectInstallMethod(t *testing.T) {
	origProbe := updateBinaryExecutableProbe
	origGobin, hadGobin := os.LookupEnv("GOBIN")
	origGopath, hadGopath := os.LookupEnv("GOPATH")
	t.Cleanup(func() {
		updateBinaryExecutableProbe = origProbe
		restoreEnv("GOBIN", origGobin, hadGobin)
		restoreEnv("GOPATH", origGopath, hadGopath)
	})

	t.Run("npm", func(t *testing.T) {
		root := t.TempDir()
		pkgRoot := filepath.Join(root, "node_modules", "@fateforge", "cnstock-cli")
		binDir := filepath.Join(pkgRoot, "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgRoot, "package.json"), []byte(`{"name":"`+npmPackageName+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		exe := filepath.Join(binDir, "cnstock-cli")
		if err := os.WriteFile(exe, []byte("bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		updateBinaryExecutableProbe = func() (string, error) { return exe, nil }
		if got := detectInstallMethod(); got != "npm" {
			t.Errorf("detectInstallMethod = %q, want npm", got)
		}
	})

	t.Run("npm_wrong_package_falls_back", func(t *testing.T) {
		root := t.TempDir()
		pkgRoot := filepath.Join(root, "node_modules", "other")
		if err := os.MkdirAll(pkgRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgRoot, "package.json"), []byte(`{"name":"other-tool"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		exe := filepath.Join(pkgRoot, "cnstock-cli")
		if err := os.WriteFile(exe, []byte("bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		updateBinaryExecutableProbe = func() (string, error) { return exe, nil }
		if got := detectInstallMethod(); got != "github-binary" {
			t.Errorf("detectInstallMethod = %q, want github-binary (foreign node_modules package)", got)
		}
	})

	t.Run("go_install_via_gobin", func(t *testing.T) {
		gobin := t.TempDir()
		_ = os.Setenv("GOBIN", gobin)
		_ = os.Unsetenv("GOPATH")
		exe := filepath.Join(gobin, "cnstock-cli")
		if err := os.WriteFile(exe, []byte("bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		updateBinaryExecutableProbe = func() (string, error) { return exe, nil }
		if got := detectInstallMethod(); got != "go-install" {
			t.Errorf("detectInstallMethod = %q, want go-install", got)
		}
	})

	t.Run("github_binary_default", func(t *testing.T) {
		_ = os.Unsetenv("GOBIN")
		_ = os.Unsetenv("GOPATH")
		dir := t.TempDir()
		exe := filepath.Join(dir, "cnstock-cli")
		if err := os.WriteFile(exe, []byte("bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		updateBinaryExecutableProbe = func() (string, error) { return exe, nil }
		if got := detectInstallMethod(); got != "github-binary" {
			t.Errorf("detectInstallMethod = %q, want github-binary", got)
		}
	})
}

func restoreEnv(key, val string, had bool) {
	if had {
		_ = os.Setenv(key, val)
	} else {
		_ = os.Unsetenv(key)
	}
}

// Fix #5: updateFailStaged derives retryable from the single output.IsRetryable
// predicate, so the code->retryable contract cannot drift from a local table.
func TestUpdateFailStaged_RetryableMatchesSingleSource(t *testing.T) {
	cases := []struct {
		code output.ErrorCode
		want bool
	}{
		{output.ErrNetwork, true},
		{output.ErrTimeout, true},
		{output.ErrRateLimit, true},
		{output.ErrServer, true},
		{output.ErrIntegrity, false},
		{output.ErrIO, false},
		{output.ErrForbidden, false},
		{output.ErrNotFound, false},
	}
	for _, c := range cases {
		if got := output.IsRetryable(c.code); got != c.want {
			t.Errorf("output.IsRetryable(%s) = %v, want %v", c.code, got, c.want)
		}
	}
}
