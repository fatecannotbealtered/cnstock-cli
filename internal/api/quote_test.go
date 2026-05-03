package api

import (
	"testing"
)

func TestParseQuoteResponse(t *testing.T) {
	// Simulated Tencent quote response for sh600519
	text := `v_sh600519="1~贵州茅台~600519~1800.00~1784.50~1790.00~12345678~6000000~6345678~1799.90~100~1799.80~200~1799.70~300~1799.60~400~1799.50~500~1800.10~100~1800.20~200~1800.30~300~1800.40~400~1800.50~500~~20240115150000~15.50~0.87~1810.00~1775.00~1800.00/12345678/22222222222~12345678~2222222222~0.50~12.50~~1810.00~1775.00~1.96~34567.89~45678.90~1.23~1962.95~1606.05~0.87~-~-~";`
	normalized := []string{"sh600519"}

	results := parseQuoteResponse(text, normalized)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	q := results[0]
	if q.Symbol != "sh600519" {
		t.Errorf("symbol = %q, want sh600519", q.Symbol)
	}
	if q.Name != "贵州茅台" {
		t.Errorf("name = %q, want 贵州茅台", q.Name)
	}
	if q.Price == nil || *q.Price != 1800.00 {
		t.Errorf("price = %v, want 1800.00", q.Price)
	}
	if q.Change == nil || *q.Change != 15.50 {
		t.Errorf("change = %v, want 15.50", q.Change)
	}
	if q.Market != "A股" {
		t.Errorf("market = %q, want A股", q.Market)
	}
}

func TestParseQuoteResponseNoData(t *testing.T) {
	text := `v_sh999999="";`
	normalized := []string{"sh999999"}

	results := parseQuoteResponse(text, normalized)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Error != "no data" {
		t.Errorf("error = %q, want 'no data'", results[0].Error)
	}
}

func TestParseQuoteResponseMissing(t *testing.T) {
	text := ``
	normalized := []string{"sh600519"}

	results := parseQuoteResponse(text, normalized)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Error == "" {
		t.Error("expected error for missing symbol")
	}
}
