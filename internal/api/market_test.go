package api

import "testing"

const breadthFixture = `{"rc":0,"data":{"total":3,"diff":[
  {"f3":-0.74,"f6":1363887868514.9,"f14":"上证指数","f104":1284,"f105":1008,"f106":60},
  {"f3":-1.33,"f6":1705304054213.09,"f14":"深证综指","f104":1742,"f105":1107,"f106":74},
  {"f3":5.59,"f6":31879947939.0,"f14":"北证50","f104":295,"f105":22,"f106":0}
]}}`

func TestParseBreadthResponse(t *testing.T) {
	stats, err := parseBreadthResponse(breadthFixture)
	if err != nil {
		t.Fatalf("parseBreadthResponse error: %v", err)
	}
	if stats.Advancing == nil || *stats.Advancing != 3321 {
		t.Errorf("Advancing = %v, want 3321", stats.Advancing)
	}
	if stats.Declining == nil || *stats.Declining != 2137 {
		t.Errorf("Declining = %v, want 2137", stats.Declining)
	}
	if stats.Flat == nil || *stats.Flat != 134 {
		t.Errorf("Flat = %v, want 134", stats.Flat)
	}
	if stats.Amount == nil {
		t.Fatal("Amount = nil, want sum of f6")
	}
	wantAmount := 1363887868514.9 + 1705304054213.09 + 31879947939.0
	if *stats.Amount != wantAmount {
		t.Errorf("Amount = %f, want %f", *stats.Amount, wantAmount)
	}
	if len(stats.Markets) != 3 {
		t.Errorf("Markets len = %d, want 3", len(stats.Markets))
	}
}

func TestParseBreadthResponseEmpty(t *testing.T) {
	if _, err := parseBreadthResponse(`{"rc":0,"data":{"diff":[]}}`); err == nil {
		t.Error("expected error for empty diff")
	}
	if _, err := parseBreadthResponse(`not json`); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
