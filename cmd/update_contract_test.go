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
)

// Single-command update contract tests (CLI-SPEC §14): a bare `update` runs the
// whole update with no confirm token; `--dry-run` is a tokenless read-only
// preview; integrity failures are non-retryable E_INTEGRITY; replace-stage local
// failures are E_IO / E_FORBIDDEN; a skill-sync failure after a successful swap
// is partial success; interrupts emit a terminal E_INTERRUPTED envelope.

// captureUpdateRun runs runUpdate with the given globals, capturing the JSON
// envelope written to os.Stdout. It restores every global and seam afterwards.
func captureUpdateRun(t *testing.T, setup func()) (map[string]any, int) {
	t.Helper()

	// Snapshot mutable globals/seams.
	origFormat, origCompact := outputFormat, compactMode
	origDryRun, origCheck := dryRunMode, updateCheckOnly
	origTarget := updateTargetVersion
	origExit := lastExit
	origSkillSync := updateSkillSync
	origApply := updateBinaryApply
	origVerify := updateVerifySignature
	origPlatform := updateBinaryPlatform
	origExe := updateBinaryExecutable
	origProbe := updateBinaryExecutableProbe
	origAPI := updateBinaryGitHubAPI
	origClient := updateBinaryHTTPClient
	origRunPM := updateRunPackageManager
	origEndpoint := os.Getenv("CNS_UPDATE_ENDPOINT")
	t.Cleanup(func() {
		outputFormat, compactMode = origFormat, origCompact
		dryRunMode, updateCheckOnly = origDryRun, origCheck
		updateTargetVersion = origTarget
		lastExit = origExit
		updateSkillSync = origSkillSync
		updateBinaryApply = origApply
		updateVerifySignature = origVerify
		updateBinaryPlatform = origPlatform
		updateBinaryExecutable = origExe
		updateBinaryExecutableProbe = origProbe
		updateBinaryGitHubAPI = origAPI
		updateBinaryHTTPClient = origClient
		updateRunPackageManager = origRunPM
		_ = os.Setenv("CNS_UPDATE_ENDPOINT", origEndpoint)
	})

	// Reset to a clean default state for the run.
	outputFormat = "json"
	compactMode = true
	dryRunMode = false
	updateCheckOnly = false
	updateTargetVersion = ""
	lastExit = ExitOK

	setup()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w

	runErr := runUpdate(updateCmd, nil)

	_ = w.Close()
	os.Stdout = origStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	if runErr != nil && !errors.Is(runErr, ErrSilent) {
		t.Fatalf("runUpdate returned unexpected error: %v", runErr)
	}

	var env map[string]any
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("envelope is not valid JSON: %v\nraw: %s", err, buf.String())
	}
	return env, lastExit
}

// newUpdateReleaseServer serves a GitHub release JSON plus a tar.gz archive and a
// checksums.txt that matches it, so performBinaryUpdate can run end to end with
// the signature verification seam stubbed.
func newUpdateReleaseServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	// Build a one-file tar.gz containing the platform binary name.
	var archive bytes.Buffer
	gzw := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gzw)
	binName := "cnstock-cli" // updateBinaryPlatform is stubbed to linux in tests
	payload := []byte("new-binary-bytes")
	_ = tw.WriteHeader(&tar.Header{Name: binName, Mode: 0o755, Size: int64(len(payload)), Typeflag: tar.TypeReg})
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
	mux.HandleFunc("/repos/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, releaseJSON())
	})
	mux.HandleFunc("/latest", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, releaseJSON())
	})
	mux.HandleFunc("/dl/archive", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archiveBytes) })
	mux.HandleFunc("/dl/checksums", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, checksums) })
	mux.HandleFunc("/dl/bundle", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"bundle":"stub"}`) })

	srv := httptest.NewServer(mux)
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

// stubUpdateSeams points the update machinery at the test server and replaces the
// signature, apply, and skill-sync seams with the supplied behavior.
func stubUpdateSeams(srv *httptest.Server, apply func(src, dst string) (updateApplyResult, error), skillSync func(ctx context.Context, repo string) error) {
	_ = os.Setenv("CNS_UPDATE_ENDPOINT", srv.URL+"/latest")
	updateBinaryGitHubAPI = srv.URL
	updateBinaryHTTPClient = srv.Client()
	updateBinaryPlatform = func() (string, string) { return "linux", "amd64" }
	// /tmp/cnstock-cli is outside node_modules and GOBIN, so detectInstallMethod
	// returns "github-binary" — these tests exercise the Sigstore path.
	updateBinaryExecutable = func() (string, error) { return "/tmp/cnstock-cli", nil }
	updateVerifySignature = func(_, _, _ string) error { return nil }
	updateBinaryApply = apply
	updateSkillSync = skillSync
	// PM seam is a no-op for github-binary tests; PM-specific tests override this.
	updateRunPackageManager = func(context.Context, string, string) error { return nil }
}

func okApply(_, dst string) (updateApplyResult, error) {
	return updateApplyResult{Status: "installed", Path: dst}, nil
}

func okSkillSync(_ context.Context, _ string) error { return nil }

// A bare `update` must perform the whole update in one call, with NO confirm
// token required.
func TestUpdate_BareExecutesWithoutToken(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, okApply, okSkillSync)
	})
	if exit != ExitOK {
		t.Fatalf("exit = %d, want 0; env: %v", exit, env)
	}
	if ok, _ := env["ok"].(bool); !ok {
		t.Fatalf("ok = false, want true; env: %v", env)
	}
	data, _ := env["data"].(map[string]any)
	if data["status"] != "updated" {
		t.Errorf("status = %v, want updated", data["status"])
	}
	if data["binary_replaced"] != true {
		t.Errorf("binary_replaced = %v, want true", data["binary_replaced"])
	}
	if data["skill_sync_status"] != "synced" {
		t.Errorf("skill_sync_status = %v, want synced", data["skill_sync_status"])
	}
	if data["signature_verified"] != true {
		t.Errorf("signature_verified = %v, want true", data["signature_verified"])
	}
	if data["signature_status"] != "verified" {
		t.Errorf("signature_status = %v, want verified", data["signature_status"])
	}
	// No confirm token may appear anywhere in the success payload.
	if _, ok := data["confirm_token"]; ok {
		t.Error("success payload must not carry confirm_token")
	}
}

// `update --dry-run` is a read-only preview and must issue NO confirm_token and
// NO expires_at.
func TestUpdate_DryRunIssuesNoToken(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, func(_, _ string) (updateApplyResult, error) {
			t.Fatal("dry-run must not replace the binary")
			return updateApplyResult{}, nil
		}, func(context.Context, string) error {
			t.Fatal("dry-run must not sync the skill")
			return nil
		})
		dryRunMode = true
	})
	if exit != ExitOK {
		t.Fatalf("exit = %d, want 0; env: %v", exit, env)
	}
	data, _ := env["data"].(map[string]any)
	if data["status"] != "dry_run" {
		t.Errorf("status = %v, want dry_run", data["status"])
	}
	if _, ok := data["confirm_token"]; ok {
		t.Error("dry-run must NOT issue a confirm_token")
	}
	if _, ok := data["expires_at"]; ok {
		t.Error("dry-run must NOT issue expires_at")
	}
	if _, ok := data["preview"]; !ok {
		t.Error("dry-run must include a preview")
	}
}

// `update --check` is a read-only probe that changes nothing and issues no token.
func TestUpdate_CheckIsReadOnly(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, func(_, _ string) (updateApplyResult, error) {
			t.Fatal("--check must not replace the binary")
			return updateApplyResult{}, nil
		}, func(context.Context, string) error {
			t.Fatal("--check must not sync the skill")
			return nil
		})
		updateCheckOnly = true
	})
	if exit != ExitOK {
		t.Fatalf("exit = %d, want 0; env: %v", exit, env)
	}
	data, _ := env["data"].(map[string]any)
	if _, ok := data["confirm_token"]; ok {
		t.Error("--check must NOT issue a confirm_token")
	}
}

// An integrity failure (signature/checksum) is non-retryable E_INTEGRITY at exit 1.
func TestUpdate_IntegrityFailureNonRetryable(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, okApply, okSkillSync)
		updateVerifySignature = func(_, _, _ string) error { return errors.New("certificate identity mismatch") }
	})
	if exit != ExitGeneric {
		t.Fatalf("exit = %d, want 1 (E_INTEGRITY); env: %v", exit, env)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "E_INTEGRITY" {
		t.Errorf("code = %v, want E_INTEGRITY", errObj["code"])
	}
	if errObj["retryable"] != false {
		t.Errorf("retryable = %v, want false", errObj["retryable"])
	}
	details, _ := errObj["details"].(map[string]any)
	if details["stage"] != updateStageVerifySignature {
		t.Errorf("stage = %v, want %s", details["stage"], updateStageVerifySignature)
	}
	if details["binary_replaced"] != false {
		t.Errorf("binary_replaced = %v, want false", details["binary_replaced"])
	}
	if details["current_version"] != version {
		t.Errorf("current_version = %v, want %s", details["current_version"], version)
	}
}

// A replace-stage permission failure maps to E_FORBIDDEN (exit 4).
func TestUpdate_ReplacePermissionIsForbidden(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, func(_, _ string) (updateApplyResult, error) {
			return updateApplyResult{}, os.ErrPermission
		}, okSkillSync)
	})
	if exit != ExitForbidden {
		t.Fatalf("exit = %d, want 4 (E_FORBIDDEN); env: %v", exit, env)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "E_FORBIDDEN" {
		t.Errorf("code = %v, want E_FORBIDDEN", errObj["code"])
	}
	details, _ := errObj["details"].(map[string]any)
	if details["stage"] != updateStageReplace || details["binary_replaced"] != false {
		t.Errorf("details = %v, want stage=replace binary_replaced=false", details)
	}
}

// A replace-stage io/disk failure maps to E_IO (exit 1).
func TestUpdate_ReplaceIOFailure(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, func(_, _ string) (updateApplyResult, error) {
			return updateApplyResult{}, errors.New("no space left on device")
		}, okSkillSync)
	})
	if exit != ExitIO {
		t.Fatalf("exit = %d, want 1 (E_IO); env: %v", exit, env)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "E_IO" {
		t.Errorf("code = %v, want E_IO", errObj["code"])
	}
	if errObj["retryable"] != false {
		t.Errorf("retryable = %v, want false", errObj["retryable"])
	}
}

// A skill-sync failure AFTER a successful swap is partial success: ok:false,
// binary_replaced:true, with a skill_sync_command for the agent to run.
func TestUpdate_SkillSyncFailureIsPartialSuccess(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, okApply, func(context.Context, string) error {
			return errors.New("npx: command not found")
		})
	})
	if exit != ExitNetwork {
		t.Fatalf("exit = %d, want 7 (retryable); env: %v", exit, env)
	}
	if ok, _ := env["ok"].(bool); ok {
		t.Fatal("partial success must report ok:false, not a clean success")
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["retryable"] != true {
		t.Errorf("retryable = %v, want true", errObj["retryable"])
	}
	details, _ := errObj["details"].(map[string]any)
	if details["binary_replaced"] != true {
		t.Errorf("binary_replaced = %v, want true (swap committed)", details["binary_replaced"])
	}
	if details["stage"] != updateStageSkillSync {
		t.Errorf("stage = %v, want skill_sync", details["stage"])
	}
	if _, ok := details["skill_sync_command"]; !ok {
		t.Error("partial success must carry skill_sync_command")
	}
	// The reported current version is the NEW version, since the binary is replaced.
	if details["current_version"] != "9.9.9" {
		t.Errorf("current_version = %v, want 9.9.9 (new binary)", details["current_version"])
	}
}

// A discover/download network failure is retryable and reported at its stage with
// the binary untouched.
func TestUpdate_DownloadFailureIsRetryable(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, okApply, okSkillSync)
		// Break the asset download by pointing the binary API at a dead host but
		// keeping the discovery endpoint alive is complex; instead serve a release
		// whose archive 404s. Simplest: close the server after release JSON is
		// fetched is racy — instead use a bad GitHub API base.
		updateBinaryGitHubAPI = "http://127.0.0.1:0"
	})
	if exit != ExitNetwork && exit != ExitTimeout {
		t.Fatalf("exit = %d, want a retryable transient code; env: %v", exit, env)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["retryable"] != true {
		t.Errorf("retryable = %v, want true", errObj["retryable"])
	}
	details, _ := errObj["details"].(map[string]any)
	if details["binary_replaced"] != false {
		t.Errorf("binary_replaced = %v, want false", details["binary_replaced"])
	}
	if details["current_version"] != version {
		t.Errorf("current_version = %v, want %s (unchanged)", details["current_version"], version)
	}
}

// A bare `update` on an npm install DRIVES npm (calls updateRunPackageManager),
// then syncs the Skill, and reports status "updated". signature_status stays
// "not_checked" (npm provenance owns integrity on this path).
func TestUpdate_NPMInstallDrivesPackageManager(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	root := t.TempDir()
	pkgRoot := filepath.Join(root, "node_modules", "@fateforge", "cnstock-cli")
	exe := filepath.Join(pkgRoot, "bin", "cnstock-cli")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgRoot, "package.json"), []byte(`{"name":"`+npmPackageName+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotMethod, gotVersion string
	var skillSynced bool

	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, func(_, _ string) (updateApplyResult, error) {
			t.Fatal("npm path must not call binary apply")
			return updateApplyResult{}, nil
		}, func(_ context.Context, _ string) error {
			skillSynced = true
			return nil
		})
		// Point the install-method probe at the npm path so detectInstallMethod
		// returns "npm"; the binary-update seam is irrelevant on this path.
		updateBinaryExecutableProbe = func() (string, error) { return exe, nil }
		updateRunPackageManager = func(_ context.Context, method, targetVer string) error {
			gotMethod, gotVersion = method, targetVer
			return nil
		}
	})
	if exit != ExitOK {
		t.Fatalf("exit = %d, want 0; env: %v", exit, env)
	}
	if gotMethod != "npm" {
		t.Fatalf("expected npm to be driven, got %q", gotMethod)
	}
	if normalizeVersion(gotVersion) != "9.9.9" {
		t.Fatalf("expected target 9.9.9 passed to npm, got %q", gotVersion)
	}
	if !skillSynced {
		t.Fatal("expected Skill sync to run after npm install")
	}
	data, _ := env["data"].(map[string]any)
	if data["status"] != "updated" {
		t.Errorf("status = %v, want updated", data["status"])
	}
	if data["skill_sync_status"] != "synced" {
		t.Errorf("skill_sync_status = %v, want synced", data["skill_sync_status"])
	}
	if data["signature_status"] != "not_checked" {
		t.Errorf("signature_status = %v, want not_checked (npm path)", data["signature_status"])
	}
}

// A bare `update` on a go-install DRIVES go install.
func TestUpdate_GoInstallDrivesPackageManager(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	gobin := t.TempDir()
	exe := filepath.Join(gobin, "cnstock-cli")
	t.Setenv("GOBIN", gobin)
	t.Setenv("GOPATH", "")

	var gotMethod string

	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, func(_, _ string) (updateApplyResult, error) {
			t.Fatal("go-install path must not call binary apply")
			return updateApplyResult{}, nil
		}, okSkillSync)
		updateBinaryExecutableProbe = func() (string, error) { return exe, nil }
		updateRunPackageManager = func(_ context.Context, method, _ string) error {
			gotMethod = method
			return nil
		}
	})
	if exit != ExitOK {
		t.Fatalf("exit = %d, want 0; env: %v", exit, env)
	}
	if gotMethod != "go-install" {
		t.Fatalf("expected go-install to be driven, got %q", gotMethod)
	}
	data, _ := env["data"].(map[string]any)
	if data["install_method"] != "go-install" {
		t.Errorf("install_method = %v, want go-install", data["install_method"])
	}
}

// --dry-run on a package-manager install is a read-only preview: it must NOT
// invoke the package manager, and must report the command it WOULD run.
func TestUpdate_PackageManagerDryRunDoesNotExecute(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	root := t.TempDir()
	pkgRoot := filepath.Join(root, "node_modules", "@fateforge", "cnstock-cli")
	exe := filepath.Join(pkgRoot, "bin", "cnstock-cli")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgRoot, "package.json"), []byte(`{"name":"`+npmPackageName+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, func(_, _ string) (updateApplyResult, error) {
			t.Fatal("dry-run must not apply binary")
			return updateApplyResult{}, nil
		}, func(_ context.Context, _ string) error {
			t.Fatal("dry-run must not sync skill")
			return nil
		})
		updateBinaryExecutableProbe = func() (string, error) { return exe, nil }
		updateRunPackageManager = func(context.Context, string, string) error {
			t.Fatal("dry-run must not invoke the package manager")
			return nil
		}
		dryRunMode = true
	})
	if exit != ExitOK {
		t.Fatalf("exit = %d, want 0; env: %v", exit, env)
	}
	data, _ := env["data"].(map[string]any)
	if data["status"] != "dry_run" {
		t.Errorf("status = %v, want dry_run", data["status"])
	}
	if _, ok := data["preview"]; !ok {
		t.Error("dry-run must include a preview")
	}
	// The preview must contain the npm install command.
	preview, _ := data["preview"].(map[string]any)
	changes, _ := preview["changes"].([]any)
	if len(changes) == 0 {
		t.Error("dry-run preview must list changes")
	}
	found := false
	for _, c := range changes {
		cm, _ := c.(map[string]any)
		if cmd, _ := cm["command"].(string); strings.Contains(cmd, npmPackageName) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected npm package command in dry-run preview: %v", data["preview"])
	}
}

// When the package manager fails, the installed binary is unchanged: report
// E_IO (exit 1), binary_replaced:false, and surface the command to run manually.
func TestUpdate_PackageManagerFailureReportsUnchanged(t *testing.T) {
	srv := newUpdateReleaseServer(t, "v9.9.9")
	root := t.TempDir()
	pkgRoot := filepath.Join(root, "node_modules", "@fateforge", "cnstock-cli")
	exe := filepath.Join(pkgRoot, "bin", "cnstock-cli")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgRoot, "package.json"), []byte(`{"name":"`+npmPackageName+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	env, exit := captureUpdateRun(t, func() {
		stubUpdateSeams(srv, okApply, okSkillSync)
		updateBinaryExecutableProbe = func() (string, error) { return exe, nil }
		updateRunPackageManager = func(context.Context, string, string) error {
			return errors.New("ETARGET no matching version")
		}
	})
	if exit != ExitIO {
		t.Fatalf("exit = %d, want %d (E_IO); env: %v", exit, ExitIO, env)
	}
	errObj, _ := env["error"].(map[string]any)
	if errObj["code"] != "E_IO" {
		t.Errorf("code = %v, want E_IO", errObj["code"])
	}
	if errObj["retryable"] != false {
		t.Errorf("retryable = %v, want false", errObj["retryable"])
	}
	details, _ := errObj["details"].(map[string]any)
	if details["binary_replaced"] != false {
		t.Errorf("binary_replaced = %v, want false", details["binary_replaced"])
	}
	if cmd, _ := details["command"].(string); !strings.Contains(cmd, npmPackageName) {
		t.Errorf("expected npm command in failure details, got %q", cmd)
	}
}
