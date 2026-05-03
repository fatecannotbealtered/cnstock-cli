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
	if err := json.Unmarshal([]byte(r.Stdout), &quotes); err != nil {
		t.Fatalf("invalid JSON output: %v\nstdout: %s", err, r.Stdout)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote in JSON, got %d", len(quotes))
	}
	if quotes[0]["symbol"] != "sh600519" {
		t.Errorf("symbol = %v, want sh600519", quotes[0]["symbol"])
	}
	if quotes[0]["name"] != "贵州茅台" {
		t.Errorf("name = %v, want 贵州茅台", quotes[0]["name"])
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
	if err := json.Unmarshal([]byte(r.Stdout), &quotes); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
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
	if err := json.Unmarshal([]byte(r.Stdout), &bars); err != nil {
		t.Fatalf("invalid JSON output: %v\nstdout: %s", err, r.Stdout)
	}
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
	if err := json.Unmarshal([]byte(r.Stdout), &ticks); err != nil {
		t.Fatalf("invalid JSON output: %v\nstdout: %s", err, r.Stdout)
	}
	if len(ticks) != 4 {
		t.Fatalf("expected 4 ticks, got %d", len(ticks))
	}
	if ticks[0]["time"] != "0930" {
		t.Errorf("tick[0].time = %v, want 0930", ticks[0]["time"])
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
	if err := json.Unmarshal([]byte(r.Stdout), &results); err != nil {
		t.Fatalf("invalid JSON output: %v\nstdout: %s", err, r.Stdout)
	}
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
	if !strings.Contains(r.Stderr, "VALIDATION_ERROR") {
		t.Errorf("stderr should contain VALIDATION_ERROR, got: %s", r.Stderr)
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
	if err := json.Unmarshal([]byte(r.Stdout), &quotes); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
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
	if !strings.Contains(r.Stderr, "VALIDATION_ERROR") {
		t.Errorf("stderr should contain VALIDATION_ERROR, got: %s", r.Stderr)
	}
}

func TestBinary_KlineInvalidAdj(t *testing.T) {
	r := runBinary(nil, "kline", "sh600519", "--adj", "invalid", "--json")

	if r.ExitCode != 2 {
		t.Errorf("exit code = %d, want 2 (validation error); stderr: %s", r.ExitCode, r.Stderr)
	}
	if !strings.Contains(r.Stderr, "VALIDATION_ERROR") {
		t.Errorf("stderr should contain VALIDATION_ERROR, got: %s", r.Stderr)
	}
}

func TestBinary_NetworkError(t *testing.T) {
	r := runBinary(map[string]string{
		"CNS_QUOTE_ENDPOINT": "http://127.0.0.1:1/q=%s",
	}, "quote", "sh600519", "--json")

	if r.ExitCode != 7 {
		t.Errorf("exit code = %d, want 7 (network error); stderr: %s", r.ExitCode, r.Stderr)
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
	if !strings.Contains(r.Stdout, "quote") {
		t.Errorf("stdout should contain 'quote' command docs, got: %s", r.Stdout)
	}
	if !strings.Contains(r.Stdout, "kline") {
		t.Errorf("stdout should contain 'kline' command docs")
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
	if err := json.Unmarshal([]byte(r.Stdout), &quotes); err != nil {
		t.Fatalf("stdout is not valid JSON with --quiet: %v\nstdout: %s", err, r.Stdout)
	}
}
