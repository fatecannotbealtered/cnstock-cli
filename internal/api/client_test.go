package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDoOnceStatusMapping verifies upstream HTTP client-error statuses map onto
// the error taxonomy so an agent can tell "bad symbol" (404) from "rate limited"
// (429) instead of every 4xx collapsing to E_NETWORK.
func TestDoOnceStatusMapping(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantAs    func(error) bool
		wantRetry bool
	}{
		{"not_found", http.StatusNotFound, func(e error) bool { var t *NotFoundError; return errors.As(e, &t) }, false},
		{"rate_limited", http.StatusTooManyRequests, func(e error) bool { var t *RateLimitError; return errors.As(e, &t) }, true},
		{"unauthorized", http.StatusUnauthorized, func(e error) bool { var t *AuthError; return errors.As(e, &t) }, false},
		{"forbidden", http.StatusForbidden, func(e error) bool { var t *ForbiddenError; return errors.As(e, &t) }, false},
		{"server", http.StatusInternalServerError, func(e error) bool { var t *ServerError; return errors.As(e, &t) }, true},
		{"teapot_other", http.StatusTeapot, func(e error) bool { var t *NetworkError; return errors.As(e, &t) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			c := &Client{http: srv.Client(), maxRetries: 0}
			_, retryable, err := c.doOnce(context.Background(), srv.URL)
			if err == nil {
				t.Fatalf("expected error for status %d", tt.status)
			}
			if !tt.wantAs(err) {
				t.Fatalf("status %d mapped to wrong error type: %v", tt.status, err)
			}
			if retryable != tt.wantRetry {
				t.Fatalf("status %d retryable = %v, want %v", tt.status, retryable, tt.wantRetry)
			}
		})
	}
}

func TestRefererForURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"tencent", "https://qt.gtimg.cn/q=sh000001", tencentReferer},
		{"eastmoney", "https://push2.eastmoney.com/api/qt/stock/get", eastmoneyReferer},
		{"eastmoney_ex", "https://push2ex.eastmoney.com/getTopicZTPool", eastmoneyReferer},
		{"invalid", "://bad-url", tencentReferer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := refererForURL(tt.url); got != tt.want {
				t.Fatalf("refererForURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestRedactURL(t *testing.T) {
	got := RedactURL("https://user:pass@example.com/path?access_token=abc&symbol=sh600519&api_key=secret")
	if got == "" {
		t.Fatal("RedactURL returned empty string")
	}
	if got == "https://user:pass@example.com/path?access_token=abc&symbol=sh600519&api_key=secret" {
		t.Fatal("RedactURL did not redact anything")
	}
	for _, leaked := range []string{"user:pass", "abc", "secret"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("redacted URL leaked %q: %s", leaked, got)
		}
	}
}

func TestRedactText(t *testing.T) {
	got := RedactText(`Get "https://user:pass@example.com/path?token=abc": dial tcp`)
	for _, leaked := range []string{"user:pass", "abc"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("redacted text leaked %q: %s", leaked, got)
		}
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("redacted text should contain marker, got: %s", got)
	}
}
