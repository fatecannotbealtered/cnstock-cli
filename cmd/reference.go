package cmd

import (
	"fmt"

	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
	"github.com/spf13/cobra"
)

var referenceCmd = &cobra.Command{
	Use:   "reference",
	Short: "Print machine-readable command, flag, schema, and exit-code reference",
	Args:  cobra.NoArgs,
	RunE:  runReference,
}

func init() {
	rootCmd.AddCommand(referenceCmd)
}

type referenceData struct {
	CLI            string                         `json:"cli"`
	SchemaVersion  string                         `json:"schema_version"`
	Version        string                         `json:"version"`
	OutputContract referenceOutputContract        `json:"output_contract"`
	GlobalFlags    []referenceFlag                `json:"global_flags"`
	Commands       []referenceCommand             `json:"commands"`
	Environment    []referenceEnv                 `json:"environment"`
	ExitCodes      []referenceExitCode            `json:"exit_codes"`
	ErrorCodes     []referenceErrorCode           `json:"error_codes"`
	Schemas        map[string]referenceDataSchema `json:"schemas"`
}

type referenceOutputContract struct {
	Stdout        string            `json:"stdout"`
	Stderr        string            `json:"stderr"`
	DefaultFormat string            `json:"default_format"`
	Formats       map[string]string `json:"formats"`
	Envelope      string            `json:"envelope"`
}

type referenceFlag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type referenceCommand struct {
	Name         string          `json:"name"`
	Usage        string          `json:"usage"`
	Description  string          `json:"description"`
	Kind         string          `json:"kind"`
	RawSupported bool            `json:"raw_supported"`
	Flags        []referenceFlag `json:"flags,omitempty"`
	DataSchema   string          `json:"data_schema,omitempty"`
}

type referenceEnv struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type referenceExitCode struct {
	Code        int    `json:"code"`
	Meaning     string `json:"meaning"`
	AgentAction string `json:"agent_action"`
}

type referenceErrorCode struct {
	Code      output.ErrorCode `json:"code"`
	ExitCode  int              `json:"exit_code"`
	Retryable bool             `json:"retryable"`
	Meaning   string           `json:"meaning"`
}

type referenceDataSchema struct {
	Shape  string   `json:"shape"`
	Fields []string `json:"fields"`
}

func runReference(cmd *cobra.Command, args []string) error {
	if outputFormat == "text" {
		fmt.Println(referenceMarkdown())
		return nil
	}
	if outputFormat == "raw" {
		output.Raw(referenceMarkdown())
		return nil
	}
	emitJSON(buildReference())
	return nil
}

func buildReference() referenceData {
	return referenceData{
		CLI:           "cnstock-cli",
		SchemaVersion: output.SchemaVersion,
		Version:       version,
		OutputContract: referenceOutputContract{
			Stdout:        "Default command results are exactly one valid JSON document; raw mode is unwrapped passthrough.",
			Stderr:        "Progress, warnings, diagnostics, and JSON error envelopes are emitted on stderr.",
			DefaultFormat: "json",
			Formats: map[string]string{
				"json": "Structured envelope for agents.",
				"text": "Human-readable tables or prose; do not parse programmatically.",
				"raw":  "Unwrapped upstream bytes/text for commands that support raw passthrough.",
			},
			Envelope: `{"ok":true,"schema_version":"1.0","data":{},"meta":{"duration_ms":0}}`,
		},
		GlobalFlags: []referenceFlag{
			{Name: "--format", Type: "enum", Default: "json", Description: "Output format: json|text|raw."},
			{Name: "--json", Type: "bool", Description: "Compatibility alias for --format json."},
			{Name: "--fields", Type: "csv", Description: "For JSON query output, keep only the listed top-level fields inside data."},
			{Name: "--compact", Type: "bool", Description: "Emit compact single-line JSON."},
			{Name: "--quiet", Type: "bool", Description: "Suppress non-result human output."},
			{Name: "--version", Type: "bool", Description: "Print version."},
			{Name: "--help", Type: "bool", Description: "Print help."},
		},
		Commands: []referenceCommand{
			{Name: "quote", Usage: "cnstock-cli quote <symbols>", Description: "Real-time quotes for A-share, HK, US stocks, and indices.", Kind: "query", RawSupported: true, DataSchema: "quote[]"},
			{Name: "kline", Usage: "cnstock-cli kline <symbol> [--period day|week|month] [--limit N] [--adj qfq|hfq|none]", Description: "Historical K-line bars.", Kind: "query", RawSupported: true, DataSchema: "kline_bar[]", Flags: []referenceFlag{
				{Name: "--period", Type: "enum", Default: "day", Description: "K-line period: day|week|month."},
				{Name: "--limit", Type: "int", Default: "20", Description: "Number of bars, 1-500."},
				{Name: "--adj", Type: "enum", Default: "qfq", Description: "Adjustment mode: qfq|hfq|none."},
			}},
			{Name: "minute", Usage: "cnstock-cli minute <symbol>", Description: "Intraday minute ticks for the current trading day.", Kind: "query", RawSupported: true, DataSchema: "minute_tick[]"},
			{Name: "search", Usage: "cnstock-cli search <keyword>", Description: "Search stocks by Chinese name, pinyin, English name, or code.", Kind: "query", RawSupported: true, DataSchema: "search_result[]"},
			{Name: "sectors", Usage: "cnstock-cli sectors [--board hy|gn|dy] [--top N] [--direction up|down]", Description: "Sector, concept, or region ranking.", Kind: "query", RawSupported: true, DataSchema: "sector[]", Flags: []referenceFlag{
				{Name: "--board", Type: "enum", Default: "hy", Description: "Board type: hy=industry, gn=concept, dy=region."},
				{Name: "--top", Type: "int", Default: "10", Description: "Number of sectors, 1-50."},
				{Name: "--direction", Type: "enum", Default: "up", Description: "Ranking direction: up|down."},
			}},
			{Name: "market", Usage: "cnstock-cli market", Description: "Whole-market breadth, turnover, and best-effort limit-up/down statistics.", Kind: "query", RawSupported: true, DataSchema: "market_stats"},
			{Name: "reference", Usage: "cnstock-cli reference", Description: "Machine-readable command, flag, schema, and exit-code reference.", Kind: "self-description", RawSupported: true, DataSchema: "reference"},
			{Name: "context", Usage: "cnstock-cli context", Description: "Runtime environment, command list, and endpoint configuration.", Kind: "self-description", DataSchema: "context"},
			{Name: "doctor", Usage: "cnstock-cli doctor", Description: "Endpoint health and latency checks. A failed check keeps the JSON envelope ok=true but exits 7.", Kind: "self-description", DataSchema: "doctor"},
			{Name: "update", Usage: "cnstock-cli update [--method auto|npm|go|github]", Description: "Check latest GitHub release and print safe update instructions. Does not modify files.", Kind: "query", RawSupported: true, DataSchema: "update_report", Flags: []referenceFlag{
				{Name: "--method", Type: "enum", Default: "auto", Description: "Preferred update method: auto|npm|go|github."},
			}},
		},
		Environment: []referenceEnv{
			{Name: "CNS_QUOTE_ENDPOINT", Description: "Quote endpoint URL, must contain %s."},
			{Name: "CNS_KLINE_ENDPOINT", Description: "K-line endpoint URL."},
			{Name: "CNS_MINUTE_ENDPOINT", Description: "Minute endpoint URL, must contain %s."},
			{Name: "CNS_SEARCH_ENDPOINT", Description: "Search endpoint URL, must contain %s."},
			{Name: "CNS_RANK_ENDPOINT", Description: "Sector ranking endpoint URL."},
			{Name: "CNS_BREADTH_ENDPOINT", Description: "Market breadth endpoint URL."},
			{Name: "CNS_LIMITUP_ENDPOINT", Description: "Limit-up pool endpoint URL, must contain %s for date."},
			{Name: "CNS_LIMITDOWN_ENDPOINT", Description: "Limit-down pool endpoint URL, must contain %s for date."},
			{Name: "CNS_UPDATE_ENDPOINT", Description: "GitHub latest-release endpoint used by update."},
		},
		ExitCodes: []referenceExitCode{
			{Code: ExitOK, Meaning: "Success", AgentAction: "Continue."},
			{Code: ExitGeneric, Meaning: "Generic error", AgentAction: "Read error envelope; do not blindly retry."},
			{Code: ExitBadArgs, Meaning: "Arguments or usage error", AgentAction: "Fix arguments."},
			{Code: ExitNotFound, Meaning: "Resource not found", AgentAction: "Do not retry without changing input."},
			{Code: ExitAuth, Meaning: "Authentication or permission failure", AgentAction: "Prompt for credentials or permissions."},
			{Code: ExitConfirmRequired, Meaning: "Confirmation token required", AgentAction: "Run dry-run, then retry with confirm token."},
			{Code: ExitConflict, Meaning: "Precondition conflict or state drift", AgentAction: "Refresh state and retry."},
			{Code: ExitTransient, Meaning: "Retryable transient error", AgentAction: "Back off and retry."},
			{Code: ExitTimeout, Meaning: "Timeout", AgentAction: "Back off and retry."},
		},
		ErrorCodes: []referenceErrorCode{
			{Code: output.ErrValidation, ExitCode: ExitBadArgs, Retryable: false, Meaning: "Invalid arguments or usage."},
			{Code: output.ErrNotFound, ExitCode: ExitNotFound, Retryable: false, Meaning: "Symbol or resource not found."},
			{Code: output.ErrAuth, ExitCode: ExitAuth, Retryable: false, Meaning: "Authentication or permission failure."},
			{Code: output.ErrServer, ExitCode: ExitTransient, Retryable: true, Meaning: "Upstream server returned an error."},
			{Code: output.ErrNetwork, ExitCode: ExitTransient, Retryable: true, Meaning: "Network or HTTP transport failure."},
			{Code: output.ErrRateLimit, ExitCode: ExitTransient, Retryable: true, Meaning: "Rate limited by upstream."},
			{Code: output.ErrTimeout, ExitCode: ExitTimeout, Retryable: true, Meaning: "Request timeout."},
			{Code: output.ErrUnknown, ExitCode: ExitGeneric, Retryable: false, Meaning: "Unexpected error."},
		},
		Schemas: map[string]referenceDataSchema{
			"quote[]":         {Shape: "array", Fields: []string{"symbol", "market", "name", "code", "price", "prev_close", "open", "volume", "time", "change", "change_pct", "high", "low", "amount", "pe_ratio", "turnover", "bid", "ask", "warnings"}},
			"kline_bar[]":     {Shape: "array", Fields: []string{"date", "open", "close", "high", "low", "volume"}},
			"minute_tick[]":   {Shape: "array", Fields: []string{"time", "price", "volume", "amount"}},
			"search_result[]": {Shape: "array", Fields: []string{"symbol", "name", "market", "pinyin"}},
			"sector[]":        {Shape: "array", Fields: []string{"code", "name", "change_pct", "change", "price", "turnover", "volume", "turnover_rate", "advance_decline", "leading_stock"}},
			"market_stats":    {Shape: "object", Fields: []string{"advancing", "declining", "flat", "limit_up", "limit_down", "amount", "markets", "warnings"}},
			"update_report":   {Shape: "object", Fields: []string{"current_version", "latest_version", "update_available", "install_method", "release_url", "recommended_action", "commands", "notes"}},
			"context":         {Shape: "object", Fields: []string{"version", "go_version", "os", "arch", "default_format", "formats", "commands", "endpoints"}},
			"doctor":          {Shape: "object", Fields: []string{"ok", "checked_at", "endpoints"}},
			"reference":       {Shape: "object", Fields: []string{"cli", "schema_version", "version", "output_contract", "global_flags", "commands", "environment", "exit_codes", "error_codes", "schemas"}},
		},
	}
}

func referenceMarkdown() string {
	return `# cnstock-cli Reference

## Output Contract

- stdout: default command results are exactly one valid JSON document.
- stderr: progress, warnings, diagnostics, and JSON error envelopes.
- Default format: ` + "`json`" + `.
- Success envelope: ` + "`" + `{"ok":true,"schema_version":"1.0","data":{},"meta":{"duration_ms":0}}` + "`" + `.
- Failure envelope: ` + "`" + `{"ok":false,"schema_version":"1.0","error":{"code":"E_BAD_ARGS","message":"...","details":{},"retryable":false}}` + "`" + `.
- Use ` + "`--format text`" + ` for human-readable output and ` + "`--format raw`" + ` for unwrapped upstream payloads.

## Global Flags

| Flag | Type | Description |
|------|------|-------------|
| --format | enum | Output format: json (default), text, raw |
| --json | bool | Compatibility alias for --format json |
| --fields | csv | Keep only the listed top-level fields inside JSON data |
| --compact | bool | Emit compact single-line JSON |
| --quiet | bool | Suppress non-result human output |
| --version | bool | Print version |
| --help | bool | Print help |

## Commands

| Command | Usage | Data schema |
|---------|-------|-------------|
| quote | cnstock-cli quote <symbols> | quote[] |
| kline | cnstock-cli kline <symbol> [--period day\|week\|month] [--limit N] [--adj qfq\|hfq\|none] | kline_bar[] |
| minute | cnstock-cli minute <symbol> | minute_tick[] |
| search | cnstock-cli search <keyword> | search_result[] |
| sectors | cnstock-cli sectors [--board hy\|gn\|dy] [--top N] [--direction up\|down] | sector[] |
| market | cnstock-cli market | market_stats |
| reference | cnstock-cli reference | reference |
| context | cnstock-cli context | context |
| doctor | cnstock-cli doctor | doctor |
| update | cnstock-cli update [--method auto\|npm\|go\|github] | update_report |

## Exit Codes

| Code | Meaning | Agent action |
|------|---------|--------------|
| 0 | Success | Continue |
| 1 | Generic error | Read error envelope |
| 2 | Arguments or usage error | Fix arguments |
| 3 | Resource not found | Do not retry without changing input |
| 4 | Authentication or permission failure | Prompt for credentials or permissions |
| 5 | Confirmation token required | Run dry-run then retry with confirm token |
| 6 | Precondition conflict or state drift | Refresh state and retry |
| 7 | Retryable transient error | Back off and retry |
| 8 | Timeout | Back off and retry |

## Error Codes

| Code | Exit | Retryable | Meaning |
|------|------|-----------|---------|
| E_BAD_ARGS | 2 | false | Invalid arguments or usage |
| E_NOT_FOUND | 3 | false | Symbol or resource not found |
| E_AUTH | 4 | false | Authentication or permission failure |
| E_SERVER | 7 | true | Upstream server error |
| E_NETWORK | 7 | true | Network or HTTP transport failure |
| E_RATE_LIMITED | 7 | true | Rate limited by upstream |
| E_TIMEOUT | 8 | true | Request timeout |
| E_UNKNOWN | 1 | false | Unexpected error |
`
}
