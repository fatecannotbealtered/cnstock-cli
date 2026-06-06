package api

import "testing"

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
