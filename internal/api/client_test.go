package api

import (
	"strings"
	"testing"
)

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
