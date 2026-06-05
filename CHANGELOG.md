# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.3] - 2026-06-05

### Added

- **`sectors` command**: Industry/concept/region board ranking via Tencent's ranking endpoint. Flags `--board hy|gn|dy`, `--top N` (1-50), `--direction up|down` (up = top gainers). Returns board name, change percent, leading stock, advance/decline counts, turnover.
- **`market` command**: Whole-market breadth statistics — advancing/declining/flat counts, limit-up/down counts, and total turnover, aggregated across the Shanghai/Shenzhen/Beijing markets. Sourced from Eastmoney web endpoints (NOT Tencent); limit-up/down are best-effort and degrade to warnings when unavailable.
- **HK & A-share index aliases**: Bare index names now resolve correctly — `hsi`→`hkHSI`, `hstech`→`hkHSTECH`, `hscei`→`hkHSCEI`, plus `csi300`/`hs300`/`chinext`/`star50`/`sse`/`szse`. Previously misread as US tickers.
- **`doctor` command**: Probes every endpoint and reports connectivity and latency; exits 7 when any endpoint is unreachable.
- **`context` command**: Prints runtime environment (version, Go/OS/arch), default format, command list, and per-endpoint configuration (env var + whether overridden).
- **`--format json|text|raw`**: Unified global output flag. `json` is the default (stable, low-token, parseable); `text` is human-readable tables; `raw` is the unwrapped upstream payload.
- **`--compact` / `--fields`**: Reduce token usage — `--compact` emits single-line JSON; `--fields a,b,c` restricts output to an ordered subset of top-level fields.
- **New endpoint overrides**: `CNS_RANK_ENDPOINT`, `CNS_BREADTH_ENDPOINT`, `CNS_LIMITUP_ENDPOINT`, `CNS_LIMITDOWN_ENDPOINT`.

### Changed

- **Default output is now JSON** for all commands (agent-first). Use `--format text` for human-readable tables.
- **`--json` is deprecated** but retained as a compatibility alias for `--format json`.

## [1.0.2] - 2026-05-06

### Fixed

- **HK/US stock kline returning null**: The `hkfqkline` and `usfqkline` endpoints return bars with mixed-type arrays (includes `{}` objects alongside strings), which caused `json.Unmarshal` into `[]string` to fail silently, skipping all bars. The parser now extracts string fields individually from mixed-type arrays.
- **US stock minute returning null**: US stock minute data only has 3 fields (time, price, volume, no amount), but the parser required at least 4. Now accepts 3+ fields with amount as optional.
- **Kline returning null for invalid symbols**: `kline` now returns `NOT_FOUND` error (exit code 4) when the symbol has no data, instead of silently returning null.

## [1.0.1] - 2026-05-06

### Fixed

- **SKILL.md encoding**: Fixed Chinese characters and emojis displayed as `?` in the AI Agent skill file.
- **reference command encoding**: Fixed Chinese characters in JSON schema examples showing as `?`.
- **Exchange prefix mapping**: Made `cnPrefixForSixDigit` explicit for all first-digit ranges (5=SH ETF, 9=SH B-shares, 1/2=SZ bonds/B-shares) instead of relying on default fallback.

### Added

- **Output unit tests**: Added tests for `runeWidth`, `isCJK`, `stripAnsi`, `truncate`, `formatFloat`, `ChangeColor`, `Table` (CJK, truncation, quiet mode), `PrintJSON`, `PrintErrorJSON`, and `hintForCode`.
- **Lint tests integrated**: Added `test/lint/lint_test.go` with `TestGofmt` and `TestGoVet` — formatting and static analysis now run as part of `go test ./...`.
- **Symbol test coverage**: Added test cases for SH ETF (5xxxxx), SH B-shares (9xxxxx), SZ bonds (1xxxxx), SZ B-shares (2xxxxx).

### Changed

- **Release workflow**: GitHub Release notes now extract only the current version's section from CHANGELOG.md instead of including the full file.

## [1.0.0] - 2026-05-03

Initial release of cnstock-cli.

### Features

- **Real-time quotes**: Batch query for A-shares, HK stocks, US stocks, indices, and funds. Includes price, change, OHLCV, bid/ask depth (5 levels for A-shares), PE ratio, turnover.
- **Historical K-line**: Daily/weekly/monthly bars with forward (qfq), backward (hfq), or no adjustment. Limit 1-500.
- **Intraday minutes**: All minute-level ticks for the current trading day.
- **Name search**: Search by Chinese name, pinyin, or English ticker across all markets.
- **AI Agent friendly**:
  - `--json` outputs machine-readable JSON; `--quiet` suppresses non-JSON stdout.
  - Typed error codes: `VALIDATION_ERROR` (exit 2), `NOT_FOUND` (exit 4), `SERVER_ERROR` (exit 7), `NETWORK_ERROR` (exit 7), `UNKNOWN_ERROR` (exit 1).
  - `reference` command: structured listing of all commands, flags, and JSON schemas.
- **Single binary**: No runtime dependencies. Cross-platform via GoReleaser (Linux/macOS/Windows, x64/arm64).
- **npm distribution**: `npm install -g @fatecannotbealtered-/cnstock-cli` with bundled AI Agent Skill.
- **Bilingual README**: English + Chinese.
- **Test suite**: 48 test cases across 3 layers — CLI smoke tests (6), API-level integration with httptest mock (28), and binary-level E2E with exec + httptest + env var endpoint injection (14).

### Documentation

- SKILL.md with JSON output schemas, error codes, and exit codes.
- Reference command for AI self-discovery.
- **Data source disclaimer**: Endpoints are from Tencent Finance public web pages, NOT an official API. No SLA, no schema contract, no rate-limit policy. For personal/research use only.
- **Non-affiliation**: Not an official Tencent product; data rights remain with their respective holders.

[Unreleased]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.0.3...HEAD
[1.0.3]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/fatecannotbealtered/cnstock-cli/releases/tag/v1.0.0
