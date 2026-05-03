package api

import (
	"testing"
)

func TestKlineDataKeys(t *testing.T) {
	tests := []struct {
		market   string
		period   string
		adjParam string
		want     []string
	}{
		{MarketCN, "day", "qfq", []string{"qfqday", "day"}},
		{MarketCN, "day", "", []string{"day"}},
		{MarketHK, "week", "hfq", []string{"hfqweek", "week"}},
		{MarketUS, "day", "", []string{"day"}},
		{MarketUS, "day", "qfq", []string{"day"}},
	}
	for _, tt := range tests {
		got := klineDataKeys(tt.market, tt.period, tt.adjParam)
		if len(got) != len(tt.want) {
			t.Errorf("klineDataKeys(%q,%q,%q) = %v, want %v", tt.market, tt.period, tt.adjParam, got, tt.want)
			continue
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("klineDataKeys[%d] = %q, want %q", i, got[i], tt.want[i])
			}
		}
	}
}

func TestParseOptionalFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
		valid bool
	}{
		{"1800.00", 1800.00, true},
		{"0", 0, true},
		{"", 0, false},
		{"abc", 0, false},
		{"-15.5", -15.5, true},
		{"1e3", 1000, true},
	}
	for _, tt := range tests {
		got := parseOptionalFloat(tt.input)
		if tt.valid {
			if got == nil {
				t.Errorf("parseOptionalFloat(%q) = nil, want %f", tt.input, tt.want)
				continue
			}
			if *got != tt.want {
				t.Errorf("parseOptionalFloat(%q) = %f, want %f", tt.input, *got, tt.want)
			}
		} else {
			if got != nil {
				t.Errorf("parseOptionalFloat(%q) = %f, want nil", tt.input, *got)
			}
		}
	}
}

func TestKlineInvalidPeriod(t *testing.T) {
	_, err := FetchKline(bg, NewClient(), "sh600519", "5min", 10, "qfq")
	if err == nil {
		t.Error("expected error for invalid period '5min'")
	}
	var ve *ValidationError
	if e, ok := err.(*ValidationError); !ok || e == nil {
		_ = ve
		t.Errorf("expected ValidationError, got %T", err)
	}
}
