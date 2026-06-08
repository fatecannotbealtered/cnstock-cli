package api

import (
	"strings"
	"testing"
	"time"
)

func TestParseQuoteTimeUTC(t *testing.T) {
	got := parseQuoteTimeUTC("20240115150000", MarketCN)
	if got != "2024-01-15T07:00:00Z" {
		t.Fatalf("parseQuoteTimeUTC = %q, want 2024-01-15T07:00:00Z", got)
	}
}

func TestParseMinuteTimeUTC(t *testing.T) {
	got := parseMinuteTimeUTC("0930", MarketCN)
	if _, err := time.Parse(time.RFC3339, got); err != nil {
		t.Fatalf("minute time is not RFC3339: %q: %v", got, err)
	}
	if !strings.Contains(got, "T01:30:00Z") {
		t.Fatalf("parseMinuteTimeUTC = %q, want UTC time containing T01:30:00Z", got)
	}
}
