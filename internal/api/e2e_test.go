package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// bg is a shorthand for context.Background() used in all tests.
var bg = context.Background()

// mockQuoteHandler returns a handler that simulates the Tencent quote endpoint.
func mockQuoteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fields := make([]string, 50)
		fields[1] = "????"
		fields[2] = "600519"
		fields[3] = "1800.00"
		fields[4] = "1784.50"
		fields[5] = "1790.00"
		fields[6] = "12345678"
		fields[7] = "6000000"
		fields[8] = "6345678"
		fields[9] = "1799.90"
		fields[10] = "100"
		fields[11] = "1799.80"
		fields[12] = "200"
		fields[13] = "1799.70"
		fields[14] = "300"
		fields[15] = "1799.60"
		fields[16] = "400"
		fields[17] = "1799.50"
		fields[18] = "500"
		fields[19] = "1800.10"
		fields[20] = "100"
		fields[21] = "1800.20"
		fields[22] = "200"
		fields[23] = "1800.30"
		fields[24] = "300"
		fields[25] = "1800.40"
		fields[26] = "400"
		fields[27] = "1800.50"
		fields[28] = "500"
		fields[30] = "20240115150000"
		fields[31] = "15.50"
		fields[32] = "0.87"
		fields[33] = "1810.00"
		fields[34] = "1775.00"
		fields[35] = "CNY"
		fields[37] = "22222222222"
		fields[38] = "0.50"
		fields[39] = "33.50"
		fields[46] = ""
		fields[48] = "1960.00"
		fields[49] = "1500.00"

		result := strings.Join(fields, "~")
		w.Header().Set("Content-Type", "text/plain; charset=gb18030")
		_, _ = w.Write([]byte("v_sh600519=\"" + result + "\";"))
	}
}

func mockKlineHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"code": 0,
			"msg":  "",
			"data": map[string]any{
				"sh600519": map[string]any{
					"qfqday": [][]string{
						{"2024-01-12", "1780.00", "1790.00", "1795.00", "1775.00", "10000000"},
						{"2024-01-13", "1790.00", "1800.00", "1810.00", "1785.00", "12000000"},
						{"2024-01-14", "1800.00", "1795.00", "1805.00", "1790.00", "9000000"},
						{"2024-01-15", "1795.00", "1810.00", "1815.00", "1792.00", "11000000"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func mockMinuteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"sh600519": map[string]any{
					"data": map[string]any{
						"data": []string{
							"0930 1790.00 1000 1790000000",
							"0931 1791.50 800 1433200000",
							"0932 1789.00 1200 2146800000",
							"0933 1792.00 600 1075200000",
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func mockSearchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(`v_hint="sh~600519~????~GZMT^sz~000858~???~WLY^hk~00700~????~TXKG";`))
	}
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/q=" || strings.HasPrefix(path, "/q="):
			mockQuoteHandler()(w, r)
		case strings.Contains(r.RequestURI, "minute/query"):
			mockMinuteHandler()(w, r)
		case path == "/s3/" || strings.HasPrefix(path, "/s3/"):
			mockSearchHandler()(w, r)
		case strings.HasPrefix(path, "/appstock/app/"):
			mockKlineHandler()(w, r)
		default:
			http.NotFound(w, r)
		}
	}))

	t.Setenv("CNS_QUOTE_ENDPOINT", server.URL+"/q=%s")
	t.Setenv("CNS_KLINE_ENDPOINT", server.URL+"/appstock/app/%s/get?param=%s")
	t.Setenv("CNS_MINUTE_ENDPOINT", server.URL+"/minute/query?code=%s")
	t.Setenv("CNS_SEARCH_ENDPOINT", server.URL+"/s3/?v=2&q=%s&t=all&c=1")

	return server
}

func TestE2E_QuoteSingleAStock(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	quotes, err := FetchQuote(bg, NewClient(), "sh600519")
	if err != nil {
		t.Fatalf("FetchQuote error: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}

	q := quotes[0]
	if q.Symbol != "sh600519" {
		t.Errorf("symbol = %q, want sh600519", q.Symbol)
	}
	if q.Market != "A股" {
		t.Errorf("market = %q, want A股", q.Market)
	}
	if q.Name != "????" {
		t.Errorf("name = %q, want ????", q.Name)
	}
	if q.Code != "600519" {
		t.Errorf("code = %q, want 600519", q.Code)
	}
	if q.Price == nil || *q.Price != 1800.00 {
		t.Errorf("price = %v, want 1800.00", q.Price)
	}
	if q.PrevClose == nil || *q.PrevClose != 1784.50 {
		t.Errorf("prev_close = %v, want 1784.50", q.PrevClose)
	}
	if q.Open == nil || *q.Open != 1790.00 {
		t.Errorf("open = %v, want 1790.00", q.Open)
	}
	if q.Change == nil || *q.Change != 15.50 {
		t.Errorf("change = %v, want 15.50", q.Change)
	}
	if q.ChangePct == nil || *q.ChangePct != 0.87 {
		t.Errorf("change_pct = %v, want 0.87", q.ChangePct)
	}
	if q.High == nil || *q.High != 1810.00 {
		t.Errorf("high = %v, want 1810.00", q.High)
	}
	if q.Low == nil || *q.Low != 1775.00 {
		t.Errorf("low = %v, want 1775.00", q.Low)
	}
	if q.Time != "20240115150000" {
		t.Errorf("time = %q, want 20240115150000", q.Time)
	}
	if q.PeRatio == nil || *q.PeRatio != 33.50 {
		t.Errorf("pe_ratio = %v, want 33.50", q.PeRatio)
	}
	if q.Turnover == nil || *q.Turnover != 0.50 {
		t.Errorf("turnover = %v, want 0.50", q.Turnover)
	}
	if len(q.Bid) != 5 {
		t.Errorf("bid depth = %d levels, want 5", len(q.Bid))
	} else {
		if q.Bid[0].Price == nil || *q.Bid[0].Price != 1799.90 {
			t.Errorf("bid[0].price = %v, want 1799.90", q.Bid[0].Price)
		}
		if q.Bid[0].Vol == nil || *q.Bid[0].Vol != 100 {
			t.Errorf("bid[0].vol = %v, want 100", q.Bid[0].Vol)
		}
	}
	if len(q.Ask) != 5 {
		t.Errorf("ask depth = %d levels, want 5", len(q.Ask))
	} else {
		if q.Ask[0].Price == nil || *q.Ask[0].Price != 1800.10 {
			t.Errorf("ask[0].price = %v, want 1800.10", q.Ask[0].Price)
		}
	}
}

func TestE2E_KlineForwardAdjusted(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	bars, err := FetchKline(bg, NewClient(), "sh600519", "day", 4, "qfq")
	if err != nil {
		t.Fatalf("FetchKline error: %v", err)
	}
	if len(bars) != 4 {
		t.Fatalf("expected 4 bars, got %d", len(bars))
	}
	b := bars[0]
	if b.Date != "2024-01-12" {
		t.Errorf("bar[0].date = %q, want 2024-01-12", b.Date)
	}
	if b.Open == nil || *b.Open != 1780.00 {
		t.Errorf("bar[0].open = %v, want 1780.00", b.Open)
	}
	if b.Close == nil || *b.Close != 1790.00 {
		t.Errorf("bar[0].close = %v, want 1790.00", b.Close)
	}
	if b.High == nil || *b.High != 1795.00 {
		t.Errorf("bar[0].high = %v, want 1795.00", b.High)
	}
	if b.Low == nil || *b.Low != 1775.00 {
		t.Errorf("bar[0].low = %v, want 1775.00", b.Low)
	}
	if b.Volume == nil || *b.Volume != 10000000 {
		t.Errorf("bar[0].volume = %v, want 10000000", b.Volume)
	}
	if bars[3].Date != "2024-01-15" {
		t.Errorf("bar[3].date = %q, want 2024-01-15", bars[3].Date)
	}
}

func TestE2E_KlineNoAdjustment(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	_, err := FetchKline(bg, NewClient(), "sh600519", "day", 4, "none")
	if err == nil {
		t.Fatal("expected error for adj=none with no matching data")
	}
	if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestE2E_Minute(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	ticks, err := FetchMinute(bg, NewClient(), "sh600519")
	if err != nil {
		t.Fatalf("FetchMinute error: %v", err)
	}
	if len(ticks) != 4 {
		t.Fatalf("expected 4 ticks, got %d", len(ticks))
	}
	first := ticks[0]
	if first.Time != "0930" {
		t.Errorf("tick[0].time = %q, want 0930", first.Time)
	}
	if first.Price == nil || *first.Price != 1790.00 {
		t.Errorf("tick[0].price = %v, want 1790.00", first.Price)
	}
	if first.Volume == nil || *first.Volume != 1000 {
		t.Errorf("tick[0].volume = %v, want 1000", first.Volume)
	}
	if first.Amount == nil || *first.Amount != 1790000000 {
		t.Errorf("tick[0].amount = %v, want 1790000000", first.Amount)
	}
	if ticks[3].Time != "0933" {
		t.Errorf("tick[3].time = %q, want 0933", ticks[3].Time)
	}
}

func TestE2E_Search(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	results, err := FetchSearch(bg, NewClient(), "??")
	if err != nil {
		t.Fatalf("FetchSearch error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	r0 := results[0]
	if r0.Symbol != "sh600519" {
		t.Errorf("result[0].symbol = %q, want sh600519", r0.Symbol)
	}
	if r0.Name != "????" {
		t.Errorf("result[0].name = %q, want ????", r0.Name)
	}
	if r0.Market != "A股(沪)" {
		t.Errorf("result[0].market = %q, want A股(沪)", r0.Market)
	}
	if r0.Pinyin != "GZMT" {
		t.Errorf("result[0].pinyin = %q, want GZMT", r0.Pinyin)
	}
	if results[1].Symbol != "sz000858" {
		t.Errorf("result[1].symbol = %q, want sz000858", results[1].Symbol)
	}
	if results[2].Symbol != "hk00700" {
		t.Errorf("result[2].symbol = %q, want hk00700", results[2].Symbol)
	}
	if results[2].Market != "港股" {
		t.Errorf("result[2].market = %q, want 港股", results[2].Market)
	}
}

func TestE2E_SearchEmpty(t *testing.T) {
	_, err := FetchSearch(bg, NewClient(), "")
	if err == nil {
		t.Error("expected error for empty keyword")
	}
}

func TestE2E_QuoteSymbolNormalization(t *testing.T) {
	server := newTestServer(t)
	defer server.Close()

	quotes, err := FetchQuote(bg, NewClient(), "600519")
	if err != nil {
		t.Fatalf("FetchQuote error: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
	if quotes[0].Name != "????" {
		t.Errorf("name = %q, want ????", quotes[0].Name)
	}
}

func TestE2E_KlineInvalidLimit(t *testing.T) {
	_, err := FetchKline(bg, NewClient(), "sh600519", "day", 0, "qfq")
	if err == nil {
		t.Error("expected error for limit=0")
	}
	_, err = FetchKline(bg, NewClient(), "sh600519", "day", 501, "qfq")
	if err == nil {
		t.Error("expected error for limit=501")
	}
}

func TestE2E_KlineInvalidPeriod(t *testing.T) {
	_, err := FetchKline(bg, NewClient(), "sh600519", "5min", 10, "qfq")
	if err == nil {
		t.Error("expected error for invalid period")
	}
}

func TestE2E_KlineInvalidSymbol(t *testing.T) {
	_, err := FetchKline(bg, NewClient(), "", "day", 10, "qfq")
	if err == nil {
		t.Error("expected error for empty symbol")
	}
}

func TestE2E_KlineInvalidAdj(t *testing.T) {
	_, err := FetchKline(bg, NewClient(), "sh600519", "day", 10, "invalid")
	if err == nil {
		t.Error("expected error for invalid adjustment")
	}
}

func TestE2E_QuoteNoData(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`v_sh999999="";`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("CNS_QUOTE_ENDPOINT", server.URL+"/q=%s")

	quotes, err := FetchQuote(bg, NewClient(), "sh600519")
	if err != nil {
		t.Fatalf("FetchQuote error: %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(quotes))
	}
	found := false
	for _, q := range quotes {
		if q.Symbol == "sh600519" && q.Error != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for sh600519 (not returned by server)")
	}
}

func TestE2E_KlineError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":1,"msg":"bad params","data":[]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("CNS_KLINE_ENDPOINT", server.URL+"/appstock/app/%s/get?param=%s")

	_, err := FetchKline(bg, NewClient(), "sh600519", "day", 10, "qfq")
	if err == nil {
		t.Error("expected error for Tencent business error response")
	}
}

func TestE2E_NetworkError(t *testing.T) {
	t.Setenv("CNS_QUOTE_ENDPOINT", "http://127.0.0.1:1/q=%s")
	c := NewClient()
	c.maxRetries = 0
	_, err := FetchQuote(bg, c, "sh600519")
	if err == nil {
		t.Error("expected network error")
	}
}

func TestE2E_ResolveEndpointOverride(t *testing.T) {
	customURL := "http://example.com/custom-quote/%s"
	t.Setenv("CNS_QUOTE_ENDPOINT", customURL)

	got := ResolveQuoteURL("sh600519")
	want := "http://example.com/custom-quote/sh600519"
	if got != want {
		t.Errorf("ResolveQuoteURL = %q, want %q", got, want)
	}
}

func TestE2E_ResolveEndpointDefault(t *testing.T) {
	t.Setenv("CNS_QUOTE_ENDPOINT", "")
	got := ResolveQuoteURL("sh600519")
	want := "https://qt.gtimg.cn/q=sh600519"
	if got != want {
		t.Errorf("ResolveQuoteURL = %q, want %q", got, want)
	}
}

func TestE2E_MinuteNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"sh999999":{"data":{"data":[]}}}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("CNS_MINUTE_ENDPOINT", server.URL+"/minute/query?code=%s")

	_, err := FetchMinute(bg, NewClient(), "sh600519")
	if err == nil {
		t.Fatal("expected NotFoundError, got nil")
	}
	if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}
