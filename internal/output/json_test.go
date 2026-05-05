package output

import (
	"bytes"
	"os"
	"strings"
	"testing"
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
	if !strings.Contains(output, `"UNKNOWN_ERROR"`) {
		t.Errorf("PrintErrorJSON should default to UNKNOWN_ERROR, got: %s", output)
	}
	if !strings.Contains(output, `"error"`) {
		t.Errorf("PrintErrorJSON should contain error field, got: %s", output)
	}
}

func TestPrintErrorJSONWithCode(t *testing.T) {
	output := captureStderr(func() {
		PrintErrorJSONWithCode("bad args", 0, ErrValidation)
	})

	if !strings.Contains(output, "bad args") {
		t.Errorf("PrintErrorJSONWithCode should contain error message, got: %s", output)
	}
	if !strings.Contains(output, `"VALIDATION_ERROR"`) {
		t.Errorf("PrintErrorJSONWithCode should contain error code, got: %s", output)
	}
	if !strings.Contains(output, `"hint"`) {
		t.Errorf("PrintErrorJSONWithCode should contain hint field, got: %s", output)
	}
	if !strings.Contains(output, "Check command arguments") {
		t.Errorf("PrintErrorJSONWithCode should contain hint text, got: %s", output)
	}
}

func TestPrintErrorJSONWithCodeNetwork(t *testing.T) {
	output := captureStderr(func() {
		PrintErrorJSONWithCode("timeout", 0, ErrNetwork)
	})

	if !strings.Contains(output, `"NETWORK_ERROR"`) {
		t.Errorf("should contain NETWORK_ERROR, got: %s", output)
	}
	if !strings.Contains(output, "network connectivity") {
		t.Errorf("should contain network hint, got: %s", output)
	}
}

func TestPrintErrorJSONWithCodeServer(t *testing.T) {
	output := captureStderr(func() {
		PrintErrorJSONWithCode("internal error", 0, ErrServer)
	})

	if !strings.Contains(output, `"SERVER_ERROR"`) {
		t.Errorf("should contain SERVER_ERROR, got: %s", output)
	}
}

func TestPrintErrorJSONWithCodeNotFound(t *testing.T) {
	output := captureStderr(func() {
		PrintErrorJSONWithCode("symbol not found", 0, ErrNotFound)
	})

	if !strings.Contains(output, `"NOT_FOUND"`) {
		t.Errorf("should contain NOT_FOUND, got: %s", output)
	}
	if !strings.Contains(output, "verify the input") {
		t.Errorf("should contain not-found hint, got: %s", output)
	}
}

func TestHintForCode(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want string
	}{
		{ErrValidation, "Check command arguments and flags"},
		{ErrNetwork, "Check network connectivity and try again"},
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
