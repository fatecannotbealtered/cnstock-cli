package api

import "testing"

const moneyFlowFixture = `{"rc":0,"data":{
  "f57":"600519","f58":"贵州茅台",
  "f135":123456789,"f136":2.5,"f137":80000000,"f139":40000000,
  "f141":-10000000,"f143":-110000000,"f148":50000000
}}`

func TestParseMoneyFlowResponse(t *testing.T) {
	mf, err := parseMoneyFlowResponse(moneyFlowFixture, "sh600519")
	if err != nil {
		t.Fatalf("parseMoneyFlowResponse error: %v", err)
	}
	if mf.Name != "贵州茅台" {
		t.Errorf("Name = %q, want 贵州茅台", mf.Name)
	}
	if mf.MainInflow == nil || *mf.MainInflow != 123456789 {
		t.Errorf("MainInflow = %v, want 123456789", mf.MainInflow)
	}
	if mf.MainInflowPct == nil || *mf.MainInflowPct != 2.5 {
		t.Errorf("MainInflowPct = %v, want 2.5", mf.MainInflowPct)
	}
	if mf.NorthboundFlow == nil || *mf.NorthboundFlow != 50000000 {
		t.Errorf("NorthboundFlow = %v, want 50000000", mf.NorthboundFlow)
	}
	if len(mf.Untrusted) == 0 {
		t.Error("expected name in _untrusted")
	}
}

func TestParseMoneyFlowNotFound(t *testing.T) {
	if _, err := parseMoneyFlowResponse(`{"rc":0,"data":null}`, "sh600519"); err == nil {
		t.Error("expected NotFoundError for data:null")
	} else if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("expected NotFoundError, got %T", err)
	}
}

func TestParseMoneyFlowBadShape(t *testing.T) {
	if _, err := parseMoneyFlowResponse(`not json`, "sh600519"); err == nil {
		t.Error("expected error for invalid JSON")
	}
	if _, err := parseMoneyFlowResponse(`{"rc":0,"data":[1,2]}`, "sh600519"); err == nil {
		t.Error("expected error for non-object data")
	}
}

func TestFetchMoneyFlowRejectsNonAShare(t *testing.T) {
	if _, err := FetchMoneyFlow(bg, NewClient(), "usAAPL"); err == nil {
		t.Error("expected validation error for US symbol")
	}
}
