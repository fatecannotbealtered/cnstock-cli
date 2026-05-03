package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// Ensure cobra state is clean for tests.
var _ = cobra.EnableCommandSorting

func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
	return buf.String(), err
}

func TestRootHelp(t *testing.T) {
	out, err := executeCommand("--help")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("cnstock-cli")) {
		t.Error("help output should contain 'cnstock-cli'")
	}
}

func TestQuoteHelp(t *testing.T) {
	out, err := executeCommand("quote", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Real-time quotes")) {
		t.Error("quote help should contain 'Real-time quotes'")
	}
}

func TestKlineHelp(t *testing.T) {
	out, err := executeCommand("kline", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Historical K-line")) {
		t.Error("kline help should contain 'Historical K-line'")
	}
}

func TestMinuteHelp(t *testing.T) {
	out, err := executeCommand("minute", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Intraday minute")) {
		t.Error("minute help should contain 'Intraday minute'")
	}
}

func TestSearchHelp(t *testing.T) {
	out, err := executeCommand("search", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("Search stocks")) {
		t.Error("search help should contain 'Search stocks'")
	}
}

func TestVersion(t *testing.T) {
	out, err := executeCommand("--version")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(out), []byte("cnstock-cli")) {
		t.Error("version output should contain 'cnstock-cli'")
	}
}

func init() {
	cobra.EnableCommandSorting = false
}
