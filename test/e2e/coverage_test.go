package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock payloads mirror the parser fixtures in internal/api/*_test.go.
const sectorRankPayload = `{"code":0,"msg":"ok","data":{"rank_list":[
  {"code":"pt01801780","hsl":"0.25","lzg":{"code":"sh601988","name":"中国银行","zd":"0.15","zdf":"2.54","zxj":"6.05"},"name":"银行","turnover":"2568545","volume":"33244000.00","zd":"51.98","zdf":"1.33","zgb":"41/42","zxj":"3954.25"}
]}}`

const breadthPayload = `{"rc":0,"data":{"total":1,"diff":[
  {"f3":-0.74,"f6":1363887868514.9,"f14":"上证指数","f104":1284,"f105":1008,"f106":60}
]}}`

const poolPayload = `{"rc":0,"data":{"tc":42}}`

func mockJSONServer(payload string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
}

func TestBinary_SectorsJSON(t *testing.T) {
	server := mockJSONServer(sectorRankPayload)
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_RANK_ENDPOINT": server.URL + "/rank?board=%s&direct=%s&count=%d",
	}, "sectors", "--board", "hy", "--top", "5", "--json")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	var sectors []map[string]any
	decodeData(t, r.Stdout, &sectors)
	if len(sectors) != 1 {
		t.Fatalf("expected 1 sector, got %d", len(sectors))
	}
	if sectors[0]["name"] != "银行" {
		t.Errorf("name = %v, want 银行", sectors[0]["name"])
	}
	untrusted, ok := sectors[0]["_untrusted"].([]any)
	if !ok || len(untrusted) == 0 {
		t.Fatalf("sector should tag external text fields as _untrusted, got: %#v", sectors[0]["_untrusted"])
	}
}

func TestBinary_SectorsInvalidBoard(t *testing.T) {
	r := runBinary(nil, "sectors", "--board", "bogus", "--json")
	if r.ExitCode != 2 {
		t.Fatalf("exit code = %d, want 2 (validation error); stderr: %s", r.ExitCode, r.Stderr)
	}
	env := decodeError(t, r.Stdout)
	if env.Error.Code != "E_VALIDATION" || env.Error.Retryable {
		t.Fatalf("error = %+v, want E_VALIDATION retryable=false", env.Error)
	}
}

func TestBinary_MarketJSON(t *testing.T) {
	breadth := mockJSONServer(breadthPayload)
	defer breadth.Close()
	pools := mockJSONServer(poolPayload)
	defer pools.Close()

	r := runBinary(map[string]string{
		"CNS_BREADTH_ENDPOINT":   breadth.URL,
		"CNS_LIMITUP_ENDPOINT":   pools.URL + "/up?date=%s",
		"CNS_LIMITDOWN_ENDPOINT": pools.URL + "/down?date=%s",
	}, "market", "--json")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	var stats map[string]any
	decodeData(t, r.Stdout, &stats)
	if stats["advancing"] != float64(1284) {
		t.Errorf("advancing = %v, want 1284", stats["advancing"])
	}
	if stats["limit_up"] != float64(42) {
		t.Errorf("limit_up = %v, want 42", stats["limit_up"])
	}
}

func TestBinary_MarketUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_BREADTH_ENDPOINT": server.URL,
	}, "market", "--json")
	if r.ExitCode == 0 {
		t.Fatal("upstream 500 should fail")
	}
}

func TestBinary_DoctorJSON(t *testing.T) {
	// One healthy mock backs all probe targets; the update-notice check is
	// opted out so doctor cannot reach GitHub from CI.
	server := mockJSONServer(`{}`)
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT":          server.URL + "/q=%s",
		"CNS_KLINE_ENDPOINT":          server.URL + "/k/%s/%s",
		"CNS_MINUTE_ENDPOINT":         server.URL + "/m/%s",
		"CNS_SEARCH_ENDPOINT":         server.URL + "/s/%s",
		"CNS_RANK_ENDPOINT":           server.URL + "/rank?board=%s&direct=%s&count=%d",
		"CNS_BREADTH_ENDPOINT":        server.URL + "/breadth",
		"CNS_LIMITUP_ENDPOINT":        server.URL + "/up?date=%s",
		"CNS_LIMITDOWN_ENDPOINT":      server.URL + "/down?date=%s",
		"CNSTOCK_CLI_NO_UPDATE_CHECK": "1",
	}, "doctor", "--json")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	var report map[string]any
	decodeData(t, r.Stdout, &report)
	checks, ok := report["checks"].([]any)
	if !ok || len(checks) == 0 {
		t.Fatalf("doctor should report checks, got: %#v", report["checks"])
	}
	for _, c := range checks {
		check, _ := c.(map[string]any)
		if check["check"] == "release_readiness" {
			return
		}
	}
	t.Error("doctor checks should include release_readiness")
}
