package api

import "testing"

const constituentsFixture = `{"rc":0,"data":{"total":2,"diff":[
  {"f12":"600519","f14":"贵州茅台","f2":1800.00,"f3":1.33,"f127":5.4},
  {"f12":"000858","f14":"五粮液","f2":150.20,"f3":-0.85}
]}}`

func TestParseConstituentsResponse(t *testing.T) {
	members, err := parseConstituentsResponse(constituentsFixture, "BK0475")
	if err != nil {
		t.Fatalf("parseConstituentsResponse error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("got %d constituents, want 2", len(members))
	}
	c := members[0]
	if c.Code != "600519" {
		t.Errorf("Code = %q, want 600519", c.Code)
	}
	if c.Name != "贵州茅台" {
		t.Errorf("Name = %q, want 贵州茅台", c.Name)
	}
	if c.ChangePct == nil || *c.ChangePct != 1.33 {
		t.Errorf("ChangePct = %v, want 1.33", c.ChangePct)
	}
	if c.Weight == nil || *c.Weight != 5.4 {
		t.Errorf("Weight = %v, want 5.4", c.Weight)
	}
	if len(c.Untrusted) == 0 {
		t.Error("expected name in _untrusted")
	}
}

func TestParseConstituentsNotFound(t *testing.T) {
	if _, err := parseConstituentsResponse(`{"rc":0,"data":null}`, "BK9999"); err == nil {
		t.Error("expected NotFoundError for data:null")
	} else if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("expected NotFoundError, got %T", err)
	}
	if _, err := parseConstituentsResponse(`{"rc":0,"data":{"diff":[]}}`, "BK9999"); err == nil {
		t.Error("expected NotFoundError for empty diff")
	}
}

func TestParseConstituentsBadShape(t *testing.T) {
	if _, err := parseConstituentsResponse(`not json`, "BK0475"); err == nil {
		t.Error("expected error for invalid JSON")
	}
	// A malformed row is skipped, not fatal.
	members, err := parseConstituentsResponse(`{"rc":0,"data":{"diff":["bad",{"f12":"600519","f14":"贵州茅台"}]}}`, "BK0475")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 1 {
		t.Errorf("got %d constituents, want 1 (malformed row skipped)", len(members))
	}
}

func TestFetchConstituentsEmptyIndex(t *testing.T) {
	if _, err := FetchConstituents(bg, NewClient(), "  "); err == nil {
		t.Error("expected validation error for empty index")
	}
}
