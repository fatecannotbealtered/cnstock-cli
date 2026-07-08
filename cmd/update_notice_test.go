package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
)

// newFailIfHitServer returns a test server that records (and fails the test) if it
// receives any request — used to prove a code path performs no network I/O.
func newFailIfHitServer(t *testing.T, called *bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		t.Errorf("unexpected network call to %s", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// enableUpdateNoticeCache points the update-notice cache at a temp HOME and turns
// off the `.test`-binary auto-disable for the duration of the test, so the
// meta-piggyback path can be exercised against a real cache file.
func enableUpdateNoticeCache(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows UserHomeDir
	t.Setenv(updateNoticeEnvOptOut, "")
	t.Setenv(updateNoticeLegacyOptOut, "")

	orig := updateNoticeAutoDisabled
	updateNoticeAutoDisabled = updateNoticeDisabled
	t.Cleanup(func() { updateNoticeAutoDisabled = orig })
}

// seedUpdateNoticeCache writes an available-update notice into the cache.
func seedUpdateNoticeCache(t *testing.T, severity string) {
	t.Helper()
	writeUpdateNoticeCache([]updateNotice{{
		Type:               "update_available",
		Severity:           severity,
		Message:            "cnstock-cli 9.9.9 is available (current 1.0.0)",
		CurrentVersion:     "1.0.0",
		LatestVersion:      "9.9.9",
		UpdateAvailable:    true,
		RecommendedCommand: "cnstock-cli update --compact",
		CheckedAt:          time.Now().UTC().Format(time.RFC3339),
		Source:             "update_check",
	}})
}

// captureMeta runs fn (which must emit a JSON envelope to stdout) and returns the
// parsed envelope. UpdateNoticesProvider is wired exactly as in production.
func captureMeta(t *testing.T, fn func()) map[string]any {
	t.Helper()
	output.UpdateNoticesProvider = cachedUpdateNoticesAsAny
	t.Cleanup(func() { output.UpdateNoticesProvider = nil })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	var env map[string]any
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("envelope is not valid JSON: %v\nraw: %s", err, buf.String())
	}
	return env
}

func metaNotices(t *testing.T, env map[string]any) ([]any, bool) {
	t.Helper()
	meta, ok := env["meta"].(map[string]any)
	if !ok {
		t.Fatalf("envelope has no meta object: %v", env)
	}
	notices, present := meta["notices"]
	if !present {
		return nil, false
	}
	arr, ok := notices.([]any)
	if !ok {
		t.Fatalf("meta.notices is not an array: %v", notices)
	}
	return arr, true
}

// meta.notices appears on an arbitrary (non-update) command when the cache holds
// an available update.
func TestMetaNotices_PresentWhenCacheHasUpdate(t *testing.T) {
	enableUpdateNoticeCache(t)
	seedUpdateNoticeCache(t, "info")

	env := captureMeta(t, func() {
		output.RenderEnvelope(map[string]any{"symbol": "sh000001"}, nil, true, 0)
	})

	arr, present := metaNotices(t, env)
	if !present {
		t.Fatalf("meta.notices should be present when cache holds an update; env: %v", env)
	}
	if len(arr) != 1 {
		t.Fatalf("meta.notices should have one notice, got %d", len(arr))
	}
	notice, _ := arr[0].(map[string]any)
	if notice["type"] != "update_available" {
		t.Errorf("notice type = %v, want update_available", notice["type"])
	}
	if notice["source"] != "cache" {
		t.Errorf("notice source = %v, want cache (read-only from cache)", notice["source"])
	}
}

// meta.notices is ABSENT when the cache is empty.
func TestMetaNotices_AbsentWhenCacheEmpty(t *testing.T) {
	enableUpdateNoticeCache(t)
	// no seed: cache file does not exist

	env := captureMeta(t, func() {
		output.RenderEnvelope(map[string]any{"symbol": "sh000001"}, nil, true, 0)
	})
	if _, present := metaNotices(t, env); present {
		t.Fatalf("meta.notices must be omitted when the cache is empty; env: %v", env)
	}
}

func TestUpdateNoticeAutoDisabledDetectsWindowsGoTestBinary(t *testing.T) {
	orig := updateNoticeAutoDisabled
	origArgs := os.Args
	t.Cleanup(func() {
		updateNoticeAutoDisabled = orig
		os.Args = origArgs
	})
	os.Args = []string{`C:\Users\me\AppData\Local\Temp\cmd.test.exe`}
	t.Setenv(updateNoticeEnvOptOut, "")
	t.Setenv(updateNoticeLegacyOptOut, "")

	if !updateNoticeAutoDisabled() {
		t.Fatal("Windows Go test binary must not write the real update notice cache")
	}
}

// meta.notices is ABSENT when the cache is expired (older than the TTL).
func TestMetaNotices_AbsentWhenCacheExpired(t *testing.T) {
	enableUpdateNoticeCache(t)
	seedUpdateNoticeCache(t, "info")

	// Rewrite the cache file with a stale checked_at past the TTL.
	path, err := updateNoticeCachePath()
	if err != nil {
		t.Fatalf("updateNoticeCachePath: %v", err)
	}
	stale := time.Now().Add(-2 * updateNoticeCacheTTL).UTC().Format(time.RFC3339)
	cache := updateNoticeCache{CheckedAt: stale, Notices: []updateNotice{{
		Type: "update_available", UpdateAvailable: true, CurrentVersion: "1.0.0", LatestVersion: "9.9.9",
	}}}
	data, _ := json.MarshalIndent(cache, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	env := captureMeta(t, func() {
		output.RenderEnvelope(map[string]any{"symbol": "sh000001"}, nil, true, 0)
	})
	if _, present := metaNotices(t, env); present {
		t.Fatalf("meta.notices must be omitted when the cache is expired; env: %v", env)
	}
}

// The meta-piggyback path makes NO network call: cachedUpdateNoticesAsAny reads
// only the local cache. We assert this via the http test seam — the update
// endpoint is pointed at a server that fails the test if hit, and the provider is
// invoked many times without any request reaching it.
func TestMetaNotices_NoNetwork(t *testing.T) {
	enableUpdateNoticeCache(t)
	seedUpdateNoticeCache(t, "info")

	// Point the update endpoint at a server that fails the test if contacted.
	called := false
	srv := newFailIfHitServer(t, &called)
	t.Setenv("CNS_UPDATE_ENDPOINT", srv.URL+"/latest")

	for i := 0; i < 3; i++ {
		env := captureMeta(t, func() {
			output.RenderEnvelope(map[string]any{"symbol": "sh000001"}, nil, true, 0)
		})
		if _, present := metaNotices(t, env); !present {
			t.Fatalf("meta.notices should be present; env: %v", env)
		}
	}
	if called {
		t.Fatal("meta-piggyback path must not make any network call")
	}
}

func TestUpdateNoticeSeverity(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    string
	}{
		// Embedded CHANGELOG has a Security entry at 1.1.5; the delta from 1.1.4
		// to 1.1.6 includes it -> warning.
		{"security entry in delta", "1.1.4", "1.1.6", "warning"},
		// Major bump regardless of changelog content -> warning.
		{"major bump", "1.5.0", "2.0.0", "warning"},
		// Plain patch with no security entry in the delta -> info.
		{"plain patch", "1.1.6", "1.1.7", "info"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := updateNoticeSeverity(tc.current, tc.latest); got != tc.want {
				t.Errorf("updateNoticeSeverity(%q, %q) = %q, want %q", tc.current, tc.latest, got, tc.want)
			}
		})
	}
}
