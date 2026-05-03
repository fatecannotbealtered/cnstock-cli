---
name: cnstock-cli
description: "Real-time quotes, K-line history, intraday minutes, and stock search for A-shares, HK stocks, US stocks via Tencent Finance web endpoints. No API key required. NOT an official API - for personal/research use only."
metadata: {"openclaw":{"emoji":"??","requires":{"bins":["cnstock-cli"]}}}
---

# cnstock-cli

Query financial data through Tencent Finance web endpoints. No API key required.

> **IMPORTANT: This is NOT an official Tencent API.** The endpoints are observed from Tencent Finance public web pages. There is no official documentation, SLA, schema contract, or rate-limit policy. Use for personal lookup and research only ? NOT for trading, commercial products, or compliance-sensitive workloads.

> Install CLI: `npm install -g @fatecannotbealtered-/cnstock-cli`
>
> Install Skill: `npx skills add fatecannotbealtered/cnstock-cli -y -g`

## Activation

Use when the user asks about stock prices, market data, K-line history, intraday charts, or needs to find a stock code by name.

## Commands

### quote ? Real-time Quotes

```bash
cnstock-cli quote <symbols> [--json]
```

- Comma-separated codes, auto-detect market:
  - `600519`, `sh600519`, `sz000858` ? A-share
  - `00700`, `hk00700` ? HK stock
  - `AAPL`, `usAAPL`, `usTSLA` ? US stock
  - `sh000001`, `sz399001` ? Index
- Output: price, change, OHLCV, bid/ask depth (5 levels for A-shares), PE ratio, turnover

### kline ? Historical K-line

```bash
cnstock-cli kline <symbol> [--period day|week|month] [--limit N] [--adj qfq|hfq|none] [--json]
```

- Default: daily, 20 bars, forward-adjusted (qfq)
- `--limit` accepts 1-500
- Use `--adj none` for unadjusted data

### minute ? Intraday Minutes

```bash
cnstock-cli minute <symbol> [--json]
```

- Returns all minute-level ticks for the current trading day
- Fields: time, price, volume, amount

### search ? Name Search

```bash
cnstock-cli search <keyword> [--json]
```

- Supports Chinese (??), pinyin (mt), English (apple)
- Returns matching stocks across all markets

### reference ? Self-description

```bash
cnstock-cli reference
```

- Prints all commands, flags, and JSON schemas

## Global Flags

- `--json` ? Output as JSON (machine-readable)
- `--quiet` ? Suppress non-JSON stdout output

## JSON Output Schemas

### Quote

```json
{
  "symbol": "sh600519",
  "market": "A?",
  "name": "????",
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
```

### K-line Bar

```json
{
  "date": "2024-01-15",
  "open": 1780.00,
  "close": 1800.00,
  "high": 1810.00,
  "low": 1775.00,
  "volume": 12345678
}
```

### Minute Tick

```json
{
  "time": "0930",
  "price": 1790.00,
  "volume": 12345,
  "amount": 22222222
}
```

### Search Result

```json
{
  "symbol": "sh600519",
  "name": "????",
  "market": "A????",
  "pinyin": "GZMT"
}
```

### Error Response

```json
{
  "error": "symbol cannot be empty",
  "errorCode": "VALIDATION_ERROR",
  "hint": "Check command arguments and flags"
}
```

## Error Codes

| Code | Exit | Meaning |
|------|------|---------|
| `VALIDATION_ERROR` | 2 | Invalid arguments or missing required params |
| `NOT_FOUND` | 4 | Symbol or resource not found |
| `SERVER_ERROR` | 7 | Backend server returned an error |
| `NETWORK_ERROR` | 7 | Connection or HTTP transport failed |
| `UNKNOWN_ERROR` | 1 | Unexpected error |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Unknown / unexpected error |
| 2 | Bad arguments / validation error |
| 4 | Resource not found |
| 7 | Network or server error |

## Environment Variables

No environment variables needed for normal use. These override default endpoints for testing/proxying:

| Variable | Description |
|----------|-------------|
| `CNS_QUOTE_ENDPOINT` | Quote endpoint URL (must contain `%s` for symbols) |
| `CNS_KLINE_ENDPOINT` | K-line endpoint URL |
| `CNS_MINUTE_ENDPOINT` | Minute endpoint URL |
| `CNS_SEARCH_ENDPOINT` | Search endpoint URL |

## Common Symbols

See `symbols.json` in the same directory for a list of 71 common stock codes across A-shares, HK stocks, US stocks, and indices.

## Notes

- No API key is required
- **NOT an official Tencent API** ? endpoints are from public web pages, may change without notice
- No formal SLA, schema contract, or rate-limit policy exists
- The CLI follows redirects and decodes both UTF-8 and GB18030 responses
- US stock K-line data may be sparse for recent dates
- Search returns up to ~10 results per query
- For automated trading, commercial use, or compliance-sensitive reporting, use a licensed data provider (Wind, Tushare, AKShare, Bloomberg, etc.)
