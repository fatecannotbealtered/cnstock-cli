package api

import "testing"

// financialsFixture mirrors the real qt.gtimg.cn quote string for sh600519,
// trimmed but preserving the valuation tail indices (37=amount万, 38=换手率,
// 39=PE, 44=流通市值亿, 45=总市值亿, 46=PB).
func financialsFixture() string {
	parts := make([]string, 50)
	parts[0] = "1"
	parts[1] = "贵州茅台"
	parts[2] = "600519"
	parts[3] = "1291.91"
	parts[37] = "647791"   // 成交额 (万) -> 6.47791e9 yuan
	parts[38] = "0.40"     // 换手率 %
	parts[39] = "19.52"    // 市盈率
	parts[44] = "16149.93" // 流通市值 (亿) -> 1.614993e12 yuan
	parts[45] = "16149.93" // 总市值 (亿)
	parts[46] = "6.03"     // 市净率
	line := ""
	for i, p := range parts {
		if i > 0 {
			line += "~"
		}
		line += p
	}
	return `v_sh600519="` + line + `";`
}

func TestParseFinancialsResponse(t *testing.T) {
	f, err := parseFinancialsResponse(financialsFixture(), "sh600519")
	if err != nil {
		t.Fatalf("parseFinancialsResponse error: %v", err)
	}
	if f.Name != "贵州茅台" {
		t.Errorf("Name = %q, want 贵州茅台", f.Name)
	}
	if f.Code != "600519" {
		t.Errorf("Code = %q, want 600519", f.Code)
	}
	if f.Price == nil || *f.Price != 1291.91 {
		t.Errorf("Price = %v, want 1291.91", f.Price)
	}
	if f.PeRatio == nil || *f.PeRatio != 19.52 {
		t.Errorf("PeRatio = %v, want 19.52", f.PeRatio)
	}
	if f.Pb == nil || *f.Pb != 6.03 {
		t.Errorf("Pb = %v, want 6.03", f.Pb)
	}
	if f.TurnoverRate == nil || *f.TurnoverRate != 0.40 {
		t.Errorf("TurnoverRate = %v, want 0.40", f.TurnoverRate)
	}
	if f.MarketCap == nil || *f.MarketCap != 16149.93*1e8 {
		t.Errorf("MarketCap = %v, want %v", f.MarketCap, 16149.93*1e8)
	}
	if f.FloatMarketCap == nil || *f.FloatMarketCap != 16149.93*1e8 {
		t.Errorf("FloatMarketCap = %v, want %v", f.FloatMarketCap, 16149.93*1e8)
	}
	if f.Amount == nil || *f.Amount != 647791*1e4 {
		t.Errorf("Amount = %v, want %v", f.Amount, 647791*1e4)
	}
	if len(f.Untrusted) == 0 {
		t.Error("expected name in _untrusted")
	}
}

func TestParseFinancialsNotFound(t *testing.T) {
	// Tencent emits the pv_none_match sentinel for an unknown symbol.
	if _, err := parseFinancialsResponse(`v_pv_none_match="1~~~";`, "sh000000"); err == nil {
		t.Error("expected NotFoundError for pv_none_match sentinel")
	} else if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("expected NotFoundError, got %T", err)
	}
	if _, err := parseFinancialsResponse(`garbage`, "sh600519"); err == nil {
		t.Error("expected NotFoundError when no quote line is present")
	}
}

func TestFetchFinancialsRejectsBadSymbol(t *testing.T) {
	if _, err := FetchFinancials(bg, NewClient(), "   "); err == nil {
		t.Error("expected validation error for empty symbol")
	}
}

func TestEastmoneySecID(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"sh600519", "1.600519", true},
		{"600519", "1.600519", true},
		{"000001", "0.000001", true},
		{"sz000001", "0.000001", true},
		{"hk00700", "", false},
		{"AAPL", "", false},
	}
	for _, tt := range tests {
		got, err := EastmoneySecID(tt.in)
		if tt.ok {
			if err != nil {
				t.Errorf("EastmoneySecID(%q) error: %v", tt.in, err)
				continue
			}
			if got != tt.want {
				t.Errorf("EastmoneySecID(%q) = %q, want %q", tt.in, got, tt.want)
			}
		} else if err == nil {
			t.Errorf("EastmoneySecID(%q) = %q, want error", tt.in, got)
		}
	}
}
