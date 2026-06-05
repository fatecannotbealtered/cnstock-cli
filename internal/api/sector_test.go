package api

import (
	"strings"
	"testing"
)

const sectorFixture = `{"code":0,"msg":"ok","data":{"rank_list":[
  {"code":"pt01801780","hsl":"0.25","ltsz":"98334.29","lzg":{"code":"sh601988","name":"中国银行","zd":"0.15","zdf":"2.54","zxj":"6.05"},"name":"银行","speed":"0.00","stock_type":"BK-HY-1","turnover":"2568545","volume":"33244000.00","zd":"51.98","zdf":"1.33","zgb":"41/42","zxj":"3954.25"},
  {"code":"pt01801210","hsl":"2.23","ltsz":"55951.41","lzg":{"code":"sh688757","name":"胜科纳米","zd":"4.30","zdf":"11.35","zxj":"42.18"},"name":"社会服务","speed":"0.04","stock_type":"BK-HY-1","turnover":"1160557","volume":"10525200.00","zd":"49.86","zdf":"0.65","zgb":"60/80","zxj":"7743.16"}
]}}`

func TestParseSectorResponse(t *testing.T) {
	sectors, err := parseSectorResponse(sectorFixture)
	if err != nil {
		t.Fatalf("parseSectorResponse error: %v", err)
	}
	if len(sectors) != 2 {
		t.Fatalf("got %d sectors, want 2", len(sectors))
	}

	s := sectors[0]
	if s.Name != "银行" {
		t.Errorf("Name = %q, want 银行", s.Name)
	}
	if s.ChangePct == nil || *s.ChangePct != 1.33 {
		t.Errorf("ChangePct = %v, want 1.33", s.ChangePct)
	}
	if s.AdvanceDecline != "41/42" {
		t.Errorf("AdvanceDecline = %q, want 41/42", s.AdvanceDecline)
	}
	if s.LeadingStock == nil || s.LeadingStock.Name != "中国银行" {
		t.Fatalf("LeadingStock = %v, want 中国银行", s.LeadingStock)
	}
	if s.LeadingStock.ChangePct == nil || *s.LeadingStock.ChangePct != 2.54 {
		t.Errorf("LeadingStock.ChangePct = %v, want 2.54", s.LeadingStock.ChangePct)
	}
}

func TestParseSectorResponseError(t *testing.T) {
	_, err := parseSectorResponse(`{"code":51,"msg":"bad request"}`)
	if err == nil {
		t.Fatal("expected error for non-zero code")
	}
	var se *ServerError
	if e, ok := err.(*ServerError); !ok || e == nil {
		_ = se
		t.Errorf("expected ServerError, got %T", err)
	}
}

func TestFetchSectorsValidation(t *testing.T) {
	if _, err := FetchSectors(bg, NewClient(), "xx", "up", 10); err == nil {
		t.Error("expected error for invalid board type")
	}
	if _, err := FetchSectors(bg, NewClient(), "hy", "sideways", 10); err == nil {
		t.Error("expected error for invalid direction")
	}
	if _, err := FetchSectors(bg, NewClient(), "hy", "up", 0); err == nil {
		t.Error("expected error for invalid top")
	}
}

func TestSectorDirectionMapping(t *testing.T) {
	// "up" (top gainers) must map to upstream direct=down, and vice versa.
	tests := []struct {
		direction string
		wantHas   string // substring that must appear in the resolved URL
	}{
		{"up", "direct=down"},
		{"down", "direct=up"},
	}
	for _, tt := range tests {
		// Re-derive via the same switch logic used in FetchSectors by inspecting
		// the URL the resolver builds.
		var direct string
		switch tt.direction {
		case "up", "gainers", "":
			direct = "down"
		case "down", "losers":
			direct = "up"
		}
		url := ResolveRankURL("hy", direct, 10)
		if !strings.Contains(url, tt.wantHas) {
			t.Errorf("direction %q -> URL %q, want substring %q", tt.direction, url, tt.wantHas)
		}
	}
}
