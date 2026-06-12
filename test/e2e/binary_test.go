package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary once for all tests.
	dir, err := os.MkdirTemp("", "tfc-e2e-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer func() { _ = os.RemoveAll(dir) }()

	name := "cnstock-cli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binaryPath = filepath.Join(dir, name)

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/cnstock-cli")
	cmd.Dir = findProjectRoot()
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// findProjectRoot walks up from the test file directory to find go.mod.
func findProjectRoot() string {
	// Use runtime caller to find the source file location
	_, src, _, _ := runtime.Caller(0)
	dir := filepath.Dir(src)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Fallback: try the working directory
	wd, _ := os.Getwd()
	return wd
}

// --- Mock Servers ---

func mockQuoteServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fields := make([]string, 50)
		fields[1] = "贵州茅台"
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

		var result string
		for i, f := range fields {
			if i > 0 {
				result += "~"
			}
			result += f
		}
		w.Header().Set("Content-Type", "text/plain; charset=gb18030")
		_, _ = w.Write([]byte("v_sh600519=\"" + result + "\";"))
	}))
}

func mockKlineServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
}

func mockMinuteServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
}

func mockSearchServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(`v_hint="sh~600519~贵州茅台~GZMT^sz~000858~五粮液~WLY^hk~00700~腾讯控股~TXKG";`))
	}))
}

// --- Helper ---

type cmdResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type jsonEnvelope struct {
	OK            bool            `json:"ok"`
	SchemaVersion string          `json:"schema_version"`
	Data          json.RawMessage `json:"data"`
	Error         *jsonError      `json:"error"`
	Meta          map[string]any  `json:"meta"`
}

type jsonError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details"`
	Retryable bool           `json:"retryable"`
}

func decodeData(t *testing.T, stdout string, out any) jsonEnvelope {
	t.Helper()
	var env jsonEnvelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("invalid JSON envelope: %v\nstdout: %s", err, stdout)
	}
	if !env.OK {
		t.Fatalf("expected ok=true envelope, got: %+v", env)
	}
	if env.SchemaVersion != "2.0" {
		t.Fatalf("schema_version = %q, want 2.0", env.SchemaVersion)
	}
	if len(env.Data) == 0 {
		t.Fatalf("missing data in envelope: %+v", env)
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		t.Fatalf("invalid envelope data: %v\nstdout: %s", err, stdout)
	}
	return env
}

// decodeError parses the failure envelope from stdout (CLI-SPEC §4).
func decodeError(t *testing.T, stdout string) jsonEnvelope {
	t.Helper()
	var env jsonEnvelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("invalid JSON error envelope: %v\nstdout: %s", err, stdout)
	}
	if env.OK || env.Error == nil {
		t.Fatalf("expected ok=false error envelope, got: %+v", env)
	}
	if env.SchemaVersion != "2.0" {
		t.Fatalf("schema_version = %q, want 2.0", env.SchemaVersion)
	}
	if _, ok := env.Meta["duration_ms"]; !ok {
		t.Fatalf("error envelope should include meta.duration_ms: %+v", env)
	}
	return env
}

func runBinary(env map[string]string, args ...string) cmdResult {
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		exitCode = -1
	}

	return cmdResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// --- Binary-Level E2E Tests ---

func TestBinary_QuoteJSON(t *testing.T) {
	server := mockQuoteServer()
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT": server.URL + "/q=%s",
	}, "quote", "sh600519", "--json")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}

	var quotes []map[string]any
	decodeData(t, r.Stdout, &quotes)
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote in JSON, got %d", len(quotes))
	}
	if quotes[0]["symbol"] != "sh600519" {
		t.Errorf("symbol = %v, want sh600519", quotes[0]["symbol"])
	}
	if quotes[0]["name"] != "贵州茅台" {
		t.Errorf("name = %v, want 贵州茅台", quotes[0]["name"])
	}
	untrusted, ok := quotes[0]["_untrusted"].([]any)
	if !ok || len(untrusted) == 0 {
		t.Fatalf("quote should tag external text fields as _untrusted, got: %#v", quotes[0]["_untrusted"])
	}
}

func TestBinary_QuoteBatchJSON(t *testing.T) {
	server := mockQuoteServer()
	defer server.Close()

	// The mock only returns sh600519 data. sz000858 gets a "not returned" entry.
	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT": server.URL + "/q=%s",
	}, "quote", "sh600519,sz000858", "--json")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}

	var quotes []map[string]any
	decodeData(t, r.Stdout, &quotes)
	// sh600519 (data) + sz000858 (missing from server) = 2
	if len(quotes) != 2 {
		t.Fatalf("expected 2 quotes in batch JSON, got %d", len(quotes))
	}
}

func TestBinary_KlineJSON(t *testing.T) {
	server := mockKlineServer()
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_KLINE_ENDPOINT": server.URL + "/appstock/app/%s/get?param=%s",
	}, "kline", "sh600519", "--limit", "4", "--json")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}

	var bars []map[string]any
	decodeData(t, r.Stdout, &bars)
	if len(bars) != 4 {
		t.Fatalf("expected 4 bars, got %d", len(bars))
	}
	if bars[0]["date"] != "2024-01-12" {
		t.Errorf("bar[0].date = %v, want 2024-01-12", bars[0]["date"])
	}
}

func TestBinary_MinuteJSON(t *testing.T) {
	server := mockMinuteServer()
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_MINUTE_ENDPOINT": server.URL + "/minute/query?code=%s",
	}, "minute", "sh600519", "--json")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}

	var ticks []map[string]any
	decodeData(t, r.Stdout, &ticks)
	if len(ticks) != 4 {
		t.Fatalf("expected 4 ticks, got %d", len(ticks))
	}
	if got, _ := ticks[0]["time"].(string); !strings.Contains(got, "T01:30:00Z") {
		t.Errorf("tick[0].time = %v, want UTC RFC3339 containing T01:30:00Z", ticks[0]["time"])
	}
}

func TestBinary_SearchJSON(t *testing.T) {
	server := mockSearchServer()
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_SEARCH_ENDPOINT": server.URL + "/s3/?v=2&q=%s&t=all&c=1",
	}, "search", "茅台", "--json")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}

	var results []map[string]any
	decodeData(t, r.Stdout, &results)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0]["symbol"] != "sh600519" {
		t.Errorf("result[0].symbol = %v, want sh600519", results[0]["symbol"])
	}
	if results[0]["name"] != "贵州茅台" {
		t.Errorf("result[0].name = %v, want 贵州茅台", results[0]["name"])
	}
}

func TestBinary_SearchEmptyKeyword(t *testing.T) {
	r := runBinary(nil, "search", "", "--json")

	if r.ExitCode != 2 {
		t.Errorf("exit code = %d, want 2 (validation error); stderr: %s", r.ExitCode, r.Stderr)
	}
	env := decodeError(t, r.Stdout)
	if env.Error.Code != "E_VALIDATION" || env.Error.Retryable {
		t.Errorf("error = %+v, want E_VALIDATION retryable=false", env.Error)
	}
}

func TestBinary_QuoteNoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`v_sh999999="";`))
	}))
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT": server.URL + "/q=%s",
	}, "quote", "sh600519", "--json")

	// Should still exit 0 even with missing data (it returns partial results)
	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}

	var quotes []map[string]any
	decodeData(t, r.Stdout, &quotes)
	// sh999999 (no data) + sh600519 (missing) = 2
	if len(quotes) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(quotes))
	}
}

func TestBinary_KlineInvalidLimit(t *testing.T) {
	r := runBinary(nil, "kline", "sh600519", "--limit", "0", "--json")

	if r.ExitCode != 2 {
		t.Errorf("exit code = %d, want 2 (validation error); stderr: %s", r.ExitCode, r.Stderr)
	}
	env := decodeError(t, r.Stdout)
	if env.Error.Code != "E_VALIDATION" || env.Error.Retryable {
		t.Errorf("error = %+v, want E_VALIDATION retryable=false", env.Error)
	}
}

func TestBinary_KlineInvalidAdj(t *testing.T) {
	r := runBinary(nil, "kline", "sh600519", "--adj", "invalid", "--json")

	if r.ExitCode != 2 {
		t.Errorf("exit code = %d, want 2 (validation error); stderr: %s", r.ExitCode, r.Stderr)
	}
	env := decodeError(t, r.Stdout)
	if env.Error.Code != "E_VALIDATION" || env.Error.Retryable {
		t.Errorf("error = %+v, want E_VALIDATION retryable=false", env.Error)
	}
}

func TestBinary_NetworkError(t *testing.T) {
	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT": "http://127.0.0.1:1/q=%s",
	}, "quote", "sh600519", "--json")

	if r.ExitCode != 7 {
		t.Errorf("exit code = %d, want 7 (network error); stderr: %s", r.ExitCode, r.Stderr)
	}
	env := decodeError(t, r.Stdout)
	if env.Error.Code != "E_NETWORK" || !env.Error.Retryable {
		t.Errorf("error = %+v, want E_NETWORK retryable=true", env.Error)
	}
}

func TestBinary_Help(t *testing.T) {
	r := runBinary(nil, "--help")

	if r.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", r.ExitCode)
	}
	if !strings.Contains(r.Stdout, "Usage") {
		t.Errorf("stdout should contain Usage, got: %s", r.Stdout)
	}
}

func TestBinary_QuoteHelp(t *testing.T) {
	r := runBinary(nil, "quote", "--help")

	if r.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", r.ExitCode)
	}
	if !strings.Contains(r.Stdout, "quote") {
		t.Errorf("stdout should contain 'quote', got: %s", r.Stdout)
	}
}

func TestBinary_Reference(t *testing.T) {
	r := runBinary(nil, "reference")

	if r.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	var ref map[string]any
	decodeData(t, r.Stdout, &ref)
	commands, ok := ref["commands"].([]any)
	if !ok || len(commands) == 0 {
		t.Fatalf("reference should contain commands, got: %#v", ref["commands"])
	}
	if !strings.Contains(r.Stdout, "quote") || !strings.Contains(r.Stdout, "kline") {
		t.Errorf("reference should contain quote and kline command docs, got: %s", r.Stdout)
	}
}

func TestBinary_DefaultFormatIsJSON(t *testing.T) {
	server := mockQuoteServer()
	defer server.Close()

	// No --json / --format flag: default output must be JSON.
	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT": server.URL + "/q=%s",
	}, "quote", "sh600519")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	var quotes []map[string]any
	decodeData(t, r.Stdout, &quotes)
}

func TestBinary_FieldsAndCompact(t *testing.T) {
	server := mockQuoteServer()
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT": server.URL + "/q=%s",
	}, "quote", "sh600519", "--compact", "--fields", "symbol,price")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	out := strings.TrimSpace(r.Stdout)
	if strings.Contains(out, "\n") {
		t.Errorf("--compact output should be single-line, got: %s", out)
	}
	var quotes []map[string]any
	env := decodeData(t, out, &quotes)
	if _, ok := env.Meta["duration_ms"]; !ok {
		t.Error("envelope should include meta.duration_ms")
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
	if _, ok := quotes[0]["symbol"]; !ok {
		t.Error("expected 'symbol' field to be present")
	}
	if _, ok := quotes[0]["name"]; ok {
		t.Error("'name' should be filtered out by --fields")
	}
}

func TestBinary_FormatText(t *testing.T) {
	server := mockQuoteServer()
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT": server.URL + "/q=%s",
	}, "quote", "sh600519", "--format", "text")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	if json.Valid([]byte(strings.TrimSpace(r.Stdout))) {
		t.Errorf("text output should not be valid JSON: %s", r.Stdout)
	}
	if !strings.Contains(r.Stdout, "贵州茅台") {
		t.Errorf("text output should contain the stock name, got: %s", r.Stdout)
	}
}

func TestBinary_InvalidFormat(t *testing.T) {
	r := runBinary(nil, "quote", "sh600519", "--format", "yaml")

	if r.ExitCode != 2 {
		t.Errorf("exit code = %d, want 2; stderr: %s", r.ExitCode, r.Stderr)
	}
	env := decodeError(t, r.Stdout)
	if env.Error.Code != "E_VALIDATION" {
		t.Errorf("error code = %s, want E_VALIDATION", env.Error.Code)
	}
}

func TestBinary_DryRunRejectedForReadOnlyCommand(t *testing.T) {
	r := runBinary(nil, "quote", "sh600519", "--dry-run")

	if r.ExitCode != 2 {
		t.Errorf("exit code = %d, want 2; stderr: %s", r.ExitCode, r.Stderr)
	}
	env := decodeError(t, r.Stdout)
	if env.Error.Code != "E_VALIDATION" {
		t.Errorf("error code = %s, want E_VALIDATION", env.Error.Code)
	}
}

func TestBinary_Changelog(t *testing.T) {
	r := runBinary(nil, "changelog", "--compact")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	var report map[string]any
	decodeData(t, r.Stdout, &report)
	if _, ok := report["entries"].([]any); !ok {
		t.Fatalf("changelog should return entries, got: %#v", report)
	}
}

func TestBinary_Context(t *testing.T) {
	r := runBinary(nil, "context")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	var ctx map[string]any
	decodeData(t, r.Stdout, &ctx)
	if ctx["default_format"] != "json" {
		t.Errorf("default_format = %v, want json", ctx["default_format"])
	}
	if _, ok := ctx["endpoints"]; !ok {
		t.Error("context should include 'endpoints'")
	}
	if ctx["risk_tier"] != "T0" {
		t.Errorf("risk_tier = %v, want T0", ctx["risk_tier"])
	}
	if credentials, ok := ctx["credentials"].(map[string]any); !ok || credentials["required"] != false {
		t.Errorf("context should report credentials.required=false, got: %#v", ctx["credentials"])
	}
}

func TestBinary_QuietMode(t *testing.T) {
	server := mockQuoteServer()
	defer server.Close()

	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT": server.URL + "/q=%s",
	}, "quote", "sh600519", "--json", "--quiet")

	if r.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", r.ExitCode, r.Stderr)
	}
	// With --quiet + --json, stdout should be clean JSON only
	var quotes []map[string]any
	decodeData(t, r.Stdout, &quotes)
}
