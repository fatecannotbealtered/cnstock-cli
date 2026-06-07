package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func captureStderr(fn func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestPrintJSON(t *testing.T) {
	type sample struct {
		Name  string `json:"name"`
		Price int    `json:"price"`
	}

	output := captureStdout(func() {
		PrintJSON(sample{Name: "test", Price: 100})
	})

	if !strings.Contains(output, `"name": "test"`) {
		t.Errorf("PrintJSON should contain field, got: %s", output)
	}
	if !strings.Contains(output, `"price": 100`) {
		t.Errorf("PrintJSON should contain price, got: %s", output)
	}
}

func TestPrintErrorJSON(t *testing.T) {
	output := captureStderr(func() {
		PrintErrorJSON("something failed")
	})

	if !strings.Contains(output, "something failed") {
		t.Errorf("PrintErrorJSON should contain error message, got: %s", output)
	}
	if !strings.Contains(output, `"E_UNKNOWN"`) {
		t.Errorf("PrintErrorJSON should default to E_UNKNOWN, got: %s", output)
	}
	var env Envelope
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("PrintErrorJSON should emit valid JSON: %v", err)
	}
	if env.OK || env.SchemaVersion != SchemaVersion || env.Error == nil {
		t.Errorf("PrintErrorJSON should emit an error envelope, got: %+v", env)
	}
}

func TestPrintErrorJSONWithCode(t *testing.T) {
	output := captureStderr(func() {
		PrintErrorJSONWithCode("bad args", 0, ErrValidation)
	})

	if !strings.Contains(output, "bad args") {
		t.Errorf("PrintErrorJSONWithCode should contain error message, got: %s", output)
	}
	if !strings.Contains(output, `"E_BAD_ARGS"`) {
		t.Errorf("PrintErrorJSONWithCode should contain error code, got: %s", output)
	}
	var env Envelope
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("invalid error envelope: %v", err)
	}
	if env.Error == nil || env.Error.Retryable {
		t.Errorf("validation errors should not be retryable, got: %+v", env.Error)
	}
}

func TestPrintErrorJSONWithCodeNetwork(t *testing.T) {
	output := captureStderr(func() {
		PrintErrorJSONWithCode("timeout", 0, ErrNetwork)
	})

	if !strings.Contains(output, `"E_NETWORK"`) {
		t.Errorf("should contain E_NETWORK, got: %s", output)
	}
	var env Envelope
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("invalid error envelope: %v", err)
	}
	if env.Error == nil || !env.Error.Retryable {
		t.Errorf("network errors should be retryable, got: %+v", env.Error)
	}
}

func TestPrintErrorJSONWithCodeServer(t *testing.T) {
	output := captureStderr(func() {
		PrintErrorJSONWithCode("internal error", 0, ErrServer)
	})

	if !strings.Contains(output, `"E_SERVER"`) {
		t.Errorf("should contain E_SERVER, got: %s", output)
	}
}

func TestPrintErrorJSONWithCodeNotFound(t *testing.T) {
	output := captureStderr(func() {
		PrintErrorJSONWithCode("symbol not found", 0, ErrNotFound)
	})

	if !strings.Contains(output, `"E_NOT_FOUND"`) {
		t.Errorf("should contain E_NOT_FOUND, got: %s", output)
	}
}

func TestRenderEnvelope(t *testing.T) {
	output := captureStdout(func() {
		RenderEnvelope(map[string]any{"symbol": "sh600519", "name": "贵州茅台", "price": 1800}, []string{"symbol", "price"}, true, 12*time.Millisecond)
	})

	var env Envelope
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("RenderEnvelope should emit valid JSON: %v\n%s", err, output)
	}
	if !env.OK || env.SchemaVersion != SchemaVersion || env.Data == nil || env.Meta == nil {
		t.Fatalf("RenderEnvelope should emit success envelope, got: %+v", env)
	}
	if env.Meta.DurationMS != 12 {
		t.Errorf("duration_ms = %d, want 12", env.Meta.DurationMS)
	}
	var data map[string]any
	if err := json.Unmarshal(*env.Data, &data); err != nil {
		t.Fatalf("invalid data: %v", err)
	}
	if _, ok := data["symbol"]; !ok {
		t.Error("filtered data should include symbol")
	}
	if _, ok := data["name"]; ok {
		t.Error("filtered data should omit name")
	}
}

func TestHintForCode(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want string
	}{
		{ErrValidation, "Check command arguments and flags"},
		{ErrNetwork, "Check network connectivity; for HTTP 5xx, the upstream provider may be unavailable, retry later"},
		{ErrServer, "Upstream server returned an error; try again later"},
		{ErrNotFound, "Symbol or resource not found; verify the input"},
		{ErrUnknown, ""},
		{ErrConfig, ""},
	}
	for _, tt := range tests {
		got := hintForCode(tt.code)
		if got != tt.want {
			t.Errorf("hintForCode(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
