package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var referenceCmd = &cobra.Command{
	Use:   "reference",
	Short: "Print all commands, subcommands, and flags in structured markdown",
	Args:  cobra.NoArgs,
	RunE:  runReference,
}

func init() {
	rootCmd.AddCommand(referenceCmd)
}

func runReference(cmd *cobra.Command, args []string) error {
	ref := `# cnstock-cli Reference

## Global Flags

| Flag | Type | Description |
|------|------|-------------|
| --json | bool | Output result as JSON |
| --quiet | bool | Suppress non-JSON stdout output |
| --version | bool | Print version |
| --help | bool | Print help |

## Environment Variables

No environment variables needed for normal use. These override default endpoints for testing/proxying:

| Variable | Description |
|----------|-------------|
| CNS_QUOTE_ENDPOINT | Quote endpoint URL (must contain %s) |
| CNS_KLINE_ENDPOINT | K-line endpoint URL |
| CNS_MINUTE_ENDPOINT | Minute endpoint URL |
| CNS_SEARCH_ENDPOINT | Search endpoint URL |

## Commands

### quote - Real-time Quotes

` + "```" + `
cnstock-cli quote <symbols> [--json]
` + "```" + `

- Comma-separated codes, auto-detect market
- ` + "`600519`" + `, ` + "`sh600519`" + `, ` + "`sz000858`" + ` -> A-share
- ` + "`00700`" + `, ` + "`hk00700`" + ` -> HK stock
- ` + "`AAPL`" + `, ` + "`usAAPL`" + ` -> US stock
- ` + "`sh000001`" + `, ` + "`sz399001`" + ` -> Index

### kline - Historical K-line

` + "```" + `
cnstock-cli kline <symbol> [--period day|week|month] [--limit N] [--adj qfq|hfq|none] [--json]
` + "```" + `

- Default: daily, 20 bars, forward-adjusted (qfq)
- ` + "`--limit`" + ` accepts 1-500

### minute - Intraday Minutes

` + "```" + `
cnstock-cli minute <symbol> [--json]
` + "```" + `

- Returns all minute-level ticks for the current trading day
- Fields: time, price, volume, amount

### search - Name Search

` + "```" + `
cnstock-cli search <keyword> [--json]
` + "```" + `

- Supports Chinese (茅台), pinyin (mt), English (apple)

### reference - Self-description

` + "```" + `
cnstock-cli reference
` + "```" + `

- Prints this reference in structured markdown

## Error Codes

| Code | Exit | Meaning |
|------|------|---------|
| VALIDATION_ERROR | 2 | Invalid arguments or missing required params |
| NOT_FOUND | 4 | Symbol or resource not found |
| SERVER_ERROR | 7 | Backend server error |
| NETWORK_ERROR | 7 | Connection failed |
| UNKNOWN_ERROR | 1 | Unexpected error (e.g. malformed upstream response) |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Unknown / unexpected error |
| 2 | Bad arguments / validation error |
| 4 | Resource not found |
| 7 | Network or server error |

## JSON Output Schemas

### Quote

` + "```json" + `
{
  "symbol": "sh600519",
  "market": "A股",
  "name": "贵州茅台",
  "code": "600519",
  "price": 1800.00,
  "prev_close": 1784.50,
  "open": 1790.00,
  "volume": 12345678,
  "time": "20240115150000",
  "change": 15.50,
  "change_pct": 0.87,
  "high": 1810.00,
  "low": 1775.00,
  "amount": 22222222222,
  "pe_ratio": 33.50,
  "turnover": 0.50,
  "bid": [{"price": 1799.90, "vol": 100}],
  "ask": [{"price": 1800.10, "vol": 100}]
}
` + "```" + `

### K-line Bar

` + "```json" + `
{
  "date": "2024-01-15",
  "open": 1780.00,
  "close": 1800.00,
  "high": 1810.00,
  "low": 1775.00,
  "volume": 12345678
}
` + "```" + `

### Minute Tick

` + "```json" + `
{
  "time": "0930",
  "price": 1790.00,
  "volume": 12345,
  "amount": 22222222
}
` + "```" + `

### Search Result

` + "```json" + `
{
  "symbol": "sh600519",
  "name": "贵州茅台",
  "market": "A股（沪）",
  "pinyin": "GZMT"
}
` + "```" + `

### Error Response

` + "```json" + `
{
  "error": "symbol cannot be empty",
  "errorCode": "VALIDATION_ERROR",
  "hint": "Check command arguments and flags"
}
` + "```" + `
`
	fmt.Println(ref)
	return nil
}
