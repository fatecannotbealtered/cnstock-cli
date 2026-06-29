package cmd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
		wantOK  bool
	}{
		{"older", "1.1.0", "v1.1.1", -1, true},
		{"equal", "v1.1.0", "1.1.0", 0, true},
		{"newer", "1.1.0", "1.0.9", 1, true},
		{"prerelease", "1.1.0", "v1.1.1-beta.1", -1, true},
		{"dev", "dev", "v1.0.4", 0, false},
		{"devel", "(devel)", "v1.0.4", 0, false},
		{"bad", "1.0", "v1.0.4", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := compareVersions(tt.current, tt.latest)
			if ok != tt.wantOK || got != tt.want {
				t.Fatalf("compareVersions(%q, %q) = (%d, %v), want (%d, %v)", tt.current, tt.latest, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// Fail-closed contract for the in-process signature gate (CLI-SPEC §14): a
// missing bundle is refused (no skip), a failing verification aborts, and only
// a successful verification yields "verified".
func TestVerifyUpdateChecksumSignature_FailClosed(t *testing.T) {
	tmp := t.TempDir()

	if _, err := verifyUpdateChecksumSignature(context.Background(), tmp+"/checksums.txt", "", tmp); err == nil {
		t.Fatal("missing signature bundle must be refused")
	} else if !strings.Contains(err.Error(), "unsigned release") {
		t.Fatalf("unexpected error for missing bundle: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"bundle":"stub"}`))
	}))
	defer srv.Close()
	origClient := updateBinaryHTTPClient
	origVerify := updateVerifySignature
	defer func() { updateBinaryHTTPClient = origClient; updateVerifySignature = origVerify }()
	updateBinaryHTTPClient = srv.Client()

	updateVerifySignature = func(_ context.Context, _, _, _ string) error { return nil }
	status, err := verifyUpdateChecksumSignature(context.Background(), tmp+"/c.txt", srv.URL+"/b.json", tmp)
	if err != nil || status != "verified" {
		t.Fatalf("expected verified, got status=%q err=%v", status, err)
	}

	updateVerifySignature = func(_ context.Context, _, _, _ string) error { return errors.New("certificate identity mismatch") }
	if _, err := verifyUpdateChecksumSignature(context.Background(), tmp+"/c.txt", srv.URL+"/b.json", tmp); err == nil {
		t.Fatal("signature verification failure must abort")
	}
}

// isIntegrityError must classify the wrapped integrity failures so the caller
// maps them to the non-retryable E_INTEGRITY code rather than a network code.
func TestIsIntegrityError(t *testing.T) {
	if !isIntegrityError(newIntegrityError(errors.New("boom"))) {
		t.Fatal("wrapped integrity error must be detected")
	}
	if isIntegrityError(errors.New("plain")) {
		t.Fatal("plain error must not be classified as integrity")
	}
}
