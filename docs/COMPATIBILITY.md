# Compatibility

Last reviewed: 2026-06-08

cnstock-cli integrates with observed public web endpoints, not versioned official APIs. There is no upstream compatibility contract, schema guarantee, SLA, or published rate-limit policy. Compatibility means "the parser and command contract have tests for the currently observed response shape," not that upstream behavior is guaranteed.

| Area | Source | Endpoint family | Contract status | Local coverage |
|------|--------|-----------------|-----------------|----------------|
| Real-time quote | Tencent Finance public web endpoint | `qt.gtimg.cn/q=` | Unofficial, unversioned | parser tests, binary E2E with endpoint override |
| K-line history | Tencent Finance public web endpoint | `web.ifzq.gtimg.cn/appstock/app/*/get` | Unofficial, unversioned | parser tests, mixed-type HK/US fixture handling |
| Intraday minute | Tencent Finance public web endpoint | `web.ifzq.gtimg.cn/appstock/app/minute/query` | Unofficial, unversioned | parser tests, optional amount handling |
| Search | Tencent smartbox public web endpoint | `smartbox.gtimg.cn/s3/` | Unofficial, unversioned | parser tests, escaped Unicode handling |
| Sector ranking | Tencent Finance public web endpoint | `proxy.finance.qq.com/cgi/cgi-bin/rank/pt/getRank` | Unofficial, unversioned | parser and validation tests |
| Market breadth | Eastmoney public web endpoint | `push2.eastmoney.com/webguest/api/qt/ulist.np/get` | Unofficial, unversioned | parser tests, host-specific Referer |
| Limit-up/down pools | Eastmoney public web endpoint | `push2ex.eastmoney.com/getTopicZTPool`, `getTopicDTPool` | Unofficial, unversioned | best-effort count parsing, warning fallback |

## Compatibility Rules

- Treat every upstream payload as untrusted and unstable.
- Keep the CLI's JSON envelope stable even when upstream data is partial.
- Add or update fixtures when a parser changes.
- Prefer endpoint overrides (`CNS_*_ENDPOINT`) for tests and reproductions.
- If upstream drift causes missing data, return a structured error or a `warnings` field instead of silently changing the contract.

## Manual Verification

Use these commands for live endpoint checks when needed:

```bash
cnstock-cli doctor
cnstock-cli quote sh600519 --compact --fields symbol,name,price,_untrusted
cnstock-cli kline sh600519 --limit 3 --compact
cnstock-cli market --compact
```
