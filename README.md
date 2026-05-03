# cnstock-cli

[![CI](https://github.com/fatecannotbealtered/cnstock-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/fatecannotbealtered/cnstock-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fatecannotbealtered/cnstock-cli)](https://goreportcard.com/report/github.com/fatecannotbealtered/cnstock-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![npm version](https://img.shields.io/npm/v/@fatecannotbealtered-/cnstock-cli.svg)](https://www.npmjs.com/package/@fatecannotbealtered-/cnstock-cli)

English | [中文](README_zh.md)

Real-time quotes, K-line history, intraday minutes, and stock search for A-shares, HK stocks, US stocks, indices, and funds — all from your terminal. Powered by Tencent Finance web endpoints.

**Not an official Tencent product. Not affiliated with, endorsed by, or sponsored by Tencent Holdings Limited.** All trademarks belong to their respective owners.

Built with Go. Single static binary. No separate runtime to install.

[Installation](#installation) · [Commands](#commands) · [JSON Output](#json-output) · [Security](#security) · [Contributing](#contributing) · [Disclaimer](#disclaimer)

## Disclaimer

**This tool is NOT based on any official Tencent Finance API.**

The endpoints used by this CLI are web endpoints observed from Tencent Finance public web pages — they are **not documented, not contracted, and not guaranteed** by Tencent. There is no official developer documentation, no published SLA, no schema contract, and no rate-limit policy.

This project is shared for **personal learning, research, and everyday productivity** only.

**Do NOT use this tool for:**
- Automated trading or investment decisions
- Commercial products or services
- Compliance-sensitive financial reporting
- High-frequency polling or scraping
- Any scenario where data accuracy or availability is critical

Endpoints may change, return different data, or become unavailable at any time without notice. The CLI includes best-effort validation (field-count checks, `warnings` on schema drift), but this is not a substitute for a proper data provider.

For commercial use, use a licensed market data provider (e.g. Wind, Tushare, AKShare, Bloomberg) with a published API and SLA.

The MIT license covers source code only. Market data retrieved through the endpoints remains the property of its respective rights holders — this tool does not grant any rights to third-party data. Users are solely responsible for how they use the data and for compliance with applicable laws.

See [SECURITY.md](SECURITY.md) for the full data source disclaimer.

## Features

| Capability | Description |
|---|---|
| ? **Real-time Quotes** | Batch query A-shares, HK stocks, US stocks with price, change, OHLCV, bid/ask depth |
| ? **K-line History** | Daily/weekly/monthly with forward/backward/no adjustment |
| ?? **Intraday Minutes** | All minute-level ticks for the current trading day |
| ? **Name Search** | Chinese, pinyin, English — find any stock code |
| ? **AI Friendly** | `--json`, `--quiet`, `reference` command for agent self-discovery |
| ? **Single Binary** | Download and run; no runtime dependencies |
| ? **Beautiful Output** | Colored tables with CJK character support |
| ? **Multi-market** | A-shares (SH/SZ/BJ), HK stocks, US stocks, indices, funds/ETFs |

## Project Structure

```
cnstock-cli/
├── cmd/
│   ├── cnstock-cli/
│   │   └── main.go                # Entry point
│   ├── root.go                    # Root command + global flags (--json, --quiet)
│   ├── quote.go                   # quote subcommand
│   ├── kline.go                   # kline subcommand
│   ├── minute.go                  # minute subcommand
│   ├── search.go                  # search subcommand
│   ├── reference.go               # reference (AI self-discovery)
│   └── cmd_test.go                # CLI smoke tests
├── internal/
│   ├── api/
│   │   ├── client.go              # HTTP client + endpoint resolution
│   │   ├── symbol.go              # Symbol normalization + market detection
│   │   ├── quote.go               # Quote response parsing
│   │   ├── kline.go               # K-line response parsing
│   │   ├── minute.go              # Minute response parsing
│   │   ├── search.go              # Search response parsing
│   │   ├── encoding.go            # UTF-8 / GB18030 decoding
│   │   ├── types.go               # Shared data types
│   │   ├── e2e_test.go            # API-level integration tests (httptest)
│   │   ├── symbol_test.go
│   │   ├── quote_test.go
│   │   └── kline_test.go
│   └── output/
│       ├── output.go              # ANSI colors + terminal detection
│       ├── json.go                # JSON output + error codes
│       └── table.go               # Table output (CJK width-aware)
├── test/
│   └── e2e/
│       └── binary_test.go         # Binary-level E2E tests (exec + httptest)
├── scripts/
│   ├── install.js                 # npm postinstall (binary download + SHA256)
│   └── run.js                     # npm bin wrapper
├── skills/
│   └── cnstock-cli/
│       ├── SKILL.md               # AI Agent skill file
│       └── symbols.json           # Common stock symbols (reference)
├── .github/
│   └── workflows/
│       ├── ci.yml                 # CI: 3 OS × Go 1.23/1.24/1.25
│       └── release.yml            # Release: test → goreleaser → npm
├── .goreleaser.yml                # Cross-platform build config
├── go.mod
├── Makefile
├── package.json                   # npm package definition
├── README.md
├── README_zh.md
├── CHANGELOG.md
├── CONTRIBUTING.md
├── SECURITY.md
└── LICENSE                        # MIT
```

## Installation

### Quick start

```bash
# Install CLI
npm install -g @fatecannotbealtered-/cnstock-cli

# Install CLI Skill
npx skills add fatecannotbealtered/cnstock-cli -y -g

# First command
cnstock-cli quote sh600519
```

### Alternative: Go install

```bash
go install github.com/fatecannotbealtered/cnstock-cli/cmd/cnstock-cli@latest
```

### Alternative: Download binary

Download from [GitHub Releases](https://github.com/fatecannotbealtered/cnstock-cli/releases) and add to your PATH.

> **Note:** The npm install requires `curl` on your system (pre-installed on macOS and most Linux distributions). On Windows or minimal environments where `curl` is unavailable, use `go install` or download the binary directly.

## Commands

### Real-time Quotes

```bash
# Single stock
cnstock-cli quote sh600519

# Batch query (comma-separated)
cnstock-cli quote 600519,hk00700,usAAPL

# JSON output
cnstock-cli quote sh600519 --json
```

Auto-detects market from symbol format:
- `600519`, `sh600519`, `sz000858` → A-share
- `00700`, `hk00700` → HK stock
- `AAPL`, `usAAPL` → US stock
- `sh000001`, `sz399001` → Index

### Historical K-line

```bash
# Daily K-line (default: 20 bars, forward-adjusted)
cnstock-cli kline sh600519

# Weekly, 50 bars, no adjustment
cnstock-cli kline sh600519 --period week --limit 50 --adj none

# JSON output
cnstock-cli kline sh600519 --json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--period` | `day` | `day`, `week`, `month` |
| `--limit` | `20` | Number of bars (1-500) |
| `--adj` | `qfq` | `qfq` (forward), `hfq` (backward), `none` |

### Intraday Minutes

```bash
cnstock-cli minute sh600519
cnstock-cli minute sh600519 --json
```

### Name Search

```bash
# Chinese
cnstock-cli search 茅台

# Pinyin
cnstock-cli search mt

# English
cnstock-cli search apple
```

### Reference

```bash
cnstock-cli reference
```

Prints all commands, flags, JSON schemas, and error codes in structured markdown — designed for AI agent self-discovery.

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON (machine-readable) |
| `--quiet` | Suppress non-JSON stdout output |
| `--version` | Print version |
| `--help` | Print help |

## JSON Output

All commands support `--json` for machine-readable output. Example:

```json
{
  "symbol": "sh600519",
  "market": "A股",
  "name": "贵州茅台",
  "price": 1800.00,
  "change": 15.50,
  "change_pct": 0.87
}
```

See `cnstock-cli reference` for complete schemas.

## Environment Variables

This CLI does not require authentication. No environment variables are needed for normal use.

For advanced use (testing, proxying, or self-hosted endpoints), the following environment variables override the default URLs:

| Variable | Default | Description |
|----------|---------|-------------|
| `CNS_QUOTE_ENDPOINT` | `https://qt.gtimg.cn/q=%s` | Quote endpoint (must contain `%s` for symbols) |
| `CNS_KLINE_ENDPOINT` | `https://web.ifzq.gtimg.cn/appstock/app/%s/get?param=%s` | K-line endpoint |
| `CNS_MINUTE_ENDPOINT` | `https://web.ifzq.gtimg.cn/appstock/app/minute/query?code=%s` | Minute endpoint |
| `CNS_SEARCH_ENDPOINT` | `https://smartbox.gtimg.cn/s3/?v=2&q=%s&t=all&c=1` | Search endpoint |

## Error Codes

| Code | Exit | Meaning |
|------|------|---------|
| `VALIDATION_ERROR` | 2 | Invalid arguments or missing required params |
| `NOT_FOUND` | 4 | Symbol or resource not found |
| `SERVER_ERROR` | 7 | Backend server returned an error |
| `NETWORK_ERROR` | 7 | Connection or HTTP transport failed |
| `UNKNOWN_ERROR` | 1 | Unexpected error (e.g. malformed upstream response) |

## Security

See [SECURITY.md](SECURITY.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
