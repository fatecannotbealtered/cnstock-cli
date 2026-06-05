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
| --format | string | Output format: json (default), text, raw |
| --compact | bool | Emit single-line JSON (lower token count) |
| --fields | strings | Restrict JSON output to an ordered subset of top-level fields |
| --quiet | bool | Suppress non-result stdout output |
| --json | bool | Deprecated alias for --format json |
| --version | bool | Print version |
| --help | bool | Print help |

Output contract: results go to stdout; errors/progress go to stderr.
Default output is JSON (stable, low-token, parseable). Use --format text for
human-readable tables, --format raw for the unwrapped upstream payload.

## Environment Variables

No environment variables needed for normal use. These override default endpoints for testing/proxying:

| Variable | Description |
|----------|-------------|
| CNS_QUOTE_ENDPOINT | Quote endpoint URL (must contain %s) |
| CNS_KLINE_ENDPOINT | K-line endpoint URL |
| CNS_MINUTE_ENDPOINT | Minute endpoint URL |
| CNS_SEARCH_ENDPOINT | Search endpoint URL |
| CNS_RANK_ENDPOINT | Sector ranking endpoint URL |
| CNS_BREADTH_ENDPOINT | Market advance/decline endpoint URL |
| CNS_LIMITUP_ENDPOINT | Limit-up pool endpoint URL (must contain %s for date) |
| CNS_LIMITDOWN_ENDPOINT | Limit-down pool endpoint URL (must contain %s for date) |

## Commands

### quote - Real-time Quotes

` + "```" + `
cnstock-cli quote <symbols>
` + "```" + `

- Comma-separated codes, auto-detect market
- ` + "`600519`" + `, ` + "`sh600519`" + `, ` + "`sz000858`" + ` -> A-share
- ` + "`00700`" + `, ` + "`hk00700`" + ` -> HK stock
- ` + "`AAPL`" + `, ` + "`usAAPL`" + ` -> US stock
- ` + "`sh000001`" + `, ` + "`sz399001`" + ` -> Index

### kline - Historical K-line

` + "```" + `
cnstock-cli kline <symbol> [--period day|week|month] [--limit N] [--adj qfq|hfq|none]
` + "```" + `

- Default: daily, 20 bars, forward-adjusted (qfq)
- ` + "`--limit`" + ` accepts 1-500

### minute - Intraday Minutes

` + "```" + `
cnstock-cli minute <symbol>
` + "```" + `

- Returns all minute-level ticks for the current trading day
- Fields: time, price, volume, amount

### search - Name Search

` + "```" + `
cnstock-cli search <keyword>
` + "```" + `

- Supports Chinese (茅台), pinyin (mt), English (apple)

### sectors - Sector/Industry Ranking

` + "```" + `
cnstock-cli sectors [--board hy|gn|dy] [--top N] [--direction up|down]
` + "```" + `

- ` + "`--board`" + `: hy=industry (default), gn=concept, dy=region
- ` + "`--top`" + `: number of sectors, 1-50 (default 10)
- ` + "`--direction`" + `: up=top gainers (default), down=top losers
- Source: Tencent Finance ranking endpoint

### market - Whole-market Statistics

` + "```" + `
cnstock-cli market
` + "```" + `

- Advancing/declining/flat counts, limit-up/down counts, total turnover
- Aggregated across Shanghai/Shenzhen/Beijing markets
- Source: Eastmoney web endpoints (NOT Tencent)
- limit_up/limit_down are best-effort and may be omitted on non-trading days (see warnings)

### reference - Self-description

` + "```" + `
cnstock-cli reference
` + "```" + `

- Prints this reference in structured markdown

### doctor - Connectivity Health Check

` + "```" + `
cnstock-cli doctor
` + "```" + `

- Probes every endpoint and reports ok/latency_ms/error per endpoint
- Exit code 7 when any endpoint is unreachable
- Lets an agent assess environment health before relying on data commands

### context - Environment Self-awareness

` + "```" + `
cnstock-cli context
` + "```" + `

- Prints version, Go/OS/arch, default format, command list, and per-endpoint
  config (env var name + whether overridden)

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

### Sector

` + "```json" + `
{
  "code": "pt01801780",
  "name": "银行",
  "change_pct": 1.33,
  "change": 51.98,
  "price": 3954.25,
  "turnover": 2568545,
  "volume": 33244000,
  "turnover_rate": 0.25,
  "advance_decline": "41/42",
  "leading_stock": {
    "code": "sh601988",
    "name": "中国银行",
    "change_pct": 2.54,
    "price": 6.05
  }
}
` + "```" + `

### Market Statistics

` + "```json" + `
{
  "advancing": 3321,
  "declining": 2137,
  "flat": 134,
  "limit_up": 73,
  "limit_down": 11,
  "amount": 3101071870666.99,
  "markets": [
    {"name": "上证指数", "advancing": 1284, "declining": 1008, "flat": 60, "amount": 1363887868514.9}
  ],
  "warnings": []
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
