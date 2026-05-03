# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/fatecannotbealtered/cnstock-cli/releases/tag/v1.0.0
