package output

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"abc", 3},
		{"你好", 4},            // 2 CJK chars × 2
		{"abc你好", 7},         // 3 ASCII + 2 CJK × 2
		{"Ａ", 2},             // fullwidth A (U+FF21)
		{"hello世界world", 14}, // 5 + 2×2 + 5
	}
	for _, tt := range tests {
		got := runeWidth(tt.input)
		if got != tt.want {
			t.Errorf("runeWidth(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestIsCJK(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', false},
		{'1', false},
		{'中', true},  // CJK Unified Ideograph
		{'あ', true},  // Hiragana
		{'ア', true},  // Katakana
		{'가', true},  // Hangul
		{'Ａ', true},  // Fullwidth Latin (U+FF21)
		{'　', true},  // Ideographic space (U+3000)
		{'①', false}, // U+2460 Enclosed Alphanumerics — not CJK
		{'x', false},
	}
	for _, tt := range tests {
		got := isCJK(tt.r)
		if got != tt.want {
			t.Errorf("isCJK(%q U+%04X) = %v, want %v", tt.r, tt.r, got, tt.want)
		}
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"\033[1mbold\033[0m", "bold"},
		{"\033[31mred\033[0m text", "red text"},
		{"\033[1;36mcyan bold\033[0m", "cyan bold"},
		{"no escapes here", "no escapes here"},
		{"", ""},
	}
	for _, tt := range tests {
		got := stripAnsi(tt.input)
		if got != tt.want {
			t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxWidth int
		want     string
	}{
		{"hello", 10, "hello"}, // no truncation needed
		{"hello", 5, "hello"},  // exact fit
		{"hello", 4, "hel…"},   // truncated
		{"hello", 3, "he…"},    // truncated more
		{"hello", 0, ""},       // zero width
		{"你好世界", 6, "你好…"},     // CJK truncation (4 chars × 2 = 8, maxWidth 6 → 2+2=4, +1 ellipsis = 5 ≤ 6)
		{"abc", 2, "a…"},       // short ASCII
		{"你好", 4, "你好"},        // CJK exact fit
		{"你好", 3, "你…"},        // CJK truncated
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxWidth)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
		}
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{15.0, "15"},
		{15.50, "15.50"},
		{0.0, "0"},
		{-3.14, "-3.14"},
		{100.0, "100"},
		{0.87, "0.87"},
	}
	for _, tt := range tests {
		got := formatFloat(tt.input)
		if got != tt.want {
			t.Errorf("formatFloat(%f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestChangeColor(t *testing.T) {
	// In test environment, noColor is true (stdout is not a TTY),
	// so colorize returns plain text.
	tests := []struct {
		input float64
		want  string
	}{
		{15.50, "+15.50"},
		{-3.14, "-3.14"},
		{0.0, "0"},
	}
	for _, tt := range tests {
		got := ChangeColor(tt.input)
		if got != tt.want {
			t.Errorf("ChangeColor(%f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestColorize(t *testing.T) {
	// With noColor=true, colorize should return the plain message.
	msg := "hello"
	got := colorize(ansiRed, msg)
	if got != msg {
		t.Errorf("colorize with noColor should return plain text, got %q", got)
	}
}

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestTable(t *testing.T) {
	Quiet = false
	defer func() { Quiet = false }()

	output := captureStdout(func() {
		Table([]string{"Name", "Value"}, [][]string{
			{"Price", "1800.000"},
			{"Change", "+15.50"},
		})
	})

	if !strings.Contains(output, "NAME") {
		t.Error("table should contain uppercase header 'NAME'")
	}
	if !strings.Contains(output, "VALUE") {
		t.Error("table should contain uppercase header 'VALUE'")
	}
	if !strings.Contains(output, "Price") {
		t.Error("table should contain row 'Price'")
	}
	if !strings.Contains(output, "1800.000") {
		t.Error("table should contain row '1800.000'")
	}
	if !strings.Contains(output, "┌") {
		t.Error("table should contain border characters")
	}
	if !strings.Contains(output, "└") {
		t.Error("table should contain bottom border")
	}
}

func TestTableQuiet(t *testing.T) {
	Quiet = true
	defer func() { Quiet = false }()

	output := captureStdout(func() {
		Table([]string{"Name", "Value"}, [][]string{{"Price", "1800"}})
	})

	if output != "" {
		t.Errorf("table should produce no output in quiet mode, got %q", output)
	}
}

func TestTableEmptyHeaders(t *testing.T) {
	Quiet = false
	defer func() { Quiet = false }()

	output := captureStdout(func() {
		Table([]string{}, [][]string{{"a", "b"}})
	})

	if output != "" {
		t.Errorf("table with empty headers should produce no output, got %q", output)
	}
}

func TestTableCJK(t *testing.T) {
	Quiet = false
	defer func() { Quiet = false }()

	output := captureStdout(func() {
		Table([]string{"Field", "Value"}, [][]string{
			{"Name", "贵州茅台"},
		})
	})

	if !strings.Contains(output, "贵州茅台") {
		t.Error("table should contain CJK text '贵州茅台'")
	}
	if !strings.Contains(output, "│") {
		t.Error("table should contain column separators")
	}
}

func TestTableTruncation(t *testing.T) {
	Quiet = false
	defer func() { Quiet = false }()

	// A very long value should be truncated when table exceeds terminal width.
	output := captureStdout(func() {
		Table([]string{"Key"}, [][]string{
			{strings.Repeat("x", 500)},
		})
	})

	if strings.Contains(output, strings.Repeat("x", 500)) {
		t.Error("table should truncate long values")
	}
	if !strings.Contains(output, "…") {
		t.Error("truncated value should end with ellipsis")
	}
}

func TestSuccess(t *testing.T) {
	Quiet = false
	defer func() { Quiet = false }()

	output := captureStdout(func() {
		Success("done")
	})
	if !strings.Contains(output, "done") {
		t.Error("Success should print message")
	}
}

func TestSuccessQuiet(t *testing.T) {
	Quiet = true
	defer func() { Quiet = false }()

	output := captureStdout(func() {
		Success("done")
	})
	if output != "" {
		t.Error("Success should be silent in quiet mode")
	}
}

func TestInfo(t *testing.T) {
	Quiet = false
	defer func() { Quiet = false }()

	output := captureStdout(func() {
		Info("info msg")
	})
	if !strings.Contains(output, "info msg") {
		t.Error("Info should print message")
	}
}

func TestInfoQuiet(t *testing.T) {
	Quiet = true
	defer func() { Quiet = false }()

	output := captureStdout(func() {
		Info("info msg")
	})
	if output != "" {
		t.Error("Info should be silent in quiet mode")
	}
}
