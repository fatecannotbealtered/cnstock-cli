package api

import "testing"

const financialsFixture = `{"rc":0,"data":{
  "f57":"600519","f58":"贵州茅台","f43":1800.00,"f116":2261000000000,"f117":2261000000000,
  "f162":33.5,"f163":34.1,"f167":9.8,"f55":42.5,"f92":430.2,"f173":1.2,"f105":31.4,
  "f183":120000000000,"f186":60000000000,"f164":91.5,"f84":1256000000,"f85":1256000000
}}`

func TestParseFinancialsResponse(t *testing.T) {
	f, err := parseFinancialsResponse(financialsFixture, "sh600519")
	if err != nil {
		t.Fatalf("parseFinancialsResponse error: %v", err)
	}
	if f.Name != "贵州茅台" {
		t.Errorf("Name = %q, want 贵州茅台", f.Name)
	}
	if f.Code != "600519" {
		t.Errorf("Code = %q, want 600519", f.Code)
	}
	if f.PeTTM == nil || *f.PeTTM != 33.5 {
		t.Errorf("PeTTM = %v, want 33.5", f.PeTTM)
	}
	if f.Pb == nil || *f.Pb != 9.8 {
		t.Errorf("Pb = %v, want 9.8", f.Pb)
	}
	if f.Roe == nil || *f.Roe != 31.4 {
		t.Errorf("Roe = %v, want 31.4", f.Roe)
	}
	if f.MarketCap == nil || *f.MarketCap != 2261000000000 {
		t.Errorf("MarketCap = %v, want 2261000000000", f.MarketCap)
	}
	if len(f.Untrusted) == 0 {
		t.Error("expected name in _untrusted")
	}
}

func TestParseFinancialsNotFound(t *testing.T) {
	if _, err := parseFinancialsResponse(`{"rc":0,"data":null}`, "sh600519"); err == nil {
		t.Error("expected NotFoundError for data:null")
	} else if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestParseFinancialsBadShape(t *testing.T) {
	if _, err := parseFinancialsResponse(`not json`, "sh600519"); err == nil {
		t.Error("expected error for invalid JSON")
	}
	if _, err := parseFinancialsResponse(`{"rc":0,"data":[1,2,3]}`, "sh600519"); err == nil {
		t.Error("expected error for non-object data")
	}
}

func TestFetchFinancialsRejectsNonAShare(t *testing.T) {
	if _, err := FetchFinancials(bg, NewClient(), "hk00700"); err == nil {
		t.Error("expected validation error for HK symbol")
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
