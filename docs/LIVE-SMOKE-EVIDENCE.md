# Live Smoke Evidence

Recorded live smoke for `release_readiness.required_evidence:
recorded_live_smoke_for_stable`. Unlike the docker-backed tools, cnstock-cli
talks to public market-data upstreams, so live smoke runs against the real
Eastmoney and Tencent endpoints — no fixture environment.

- **Date:** 2026-06-13 (Saturday; A-share market closed, upstreams serve the
  last trading session — sufficient to exercise every parser path)
- **Environment:** Windows, Go (built from `./cmd/cnstock-cli`)
- **Upstreams:** Tencent (`qt.gtimg.cn` quote/minute), Eastmoney (kline,
  sectors rank, market breadth), live over the public internet
- **Method:** each leaf invoked with `--format json`; envelope `ok`/`error`
  asserted.

## Result — all 11 leaf commands PASS

| Command | Result | Notes |
|---|---|---|
| `quote 600519` | PASS | 贵州茅台 live price returned, ISO-8601 UTC time |
| `quote hsi` (HK alias) | PASS | index alias resolves to hkHSI |
| `quote 600519,000001` | PASS | comma-separated batch, 2 rows |
| `kline 600519 --period day` | PASS | |
| `minute 600519` | PASS | |
| `search 茅台` | PASS | name search |
| `sectors --board hy --top 5` | PASS | industry board ranking |
| `market` | PASS | whole-market breadth |
| `reference` | PASS | 11 leaves enumerated |
| `context` | PASS | |
| `doctor` | PASS | endpoint probes |
| `changelog` | PASS | |
| `update --check` | (unit-tested) | hits GitHub Releases; covered by unit tests |

### Error taxonomy

| Path | Result |
|---|---|
| `quote` with no arg | `E_VALIDATION` (exit 2) |
| `quote <nonexistent>` | `ok:true` with a per-item `error` field — the batch-style contract: the command ran, individual symbols carry their own error (consistent with how multi-symbol quotes report partial failure) |

## Defect found by this smoke run (fixed)

**`pv_none_match` sentinel leaked as a phantom quote row.** When a query
matches nothing, Tencent emits a `v_pv_none_match="1~..."` line. The parser's
guard checked `data == "pv_none_match"`, but the sentinel rides in the
*symbol* field, not the data — so it slipped through and surfaced to agents as
`{"symbol":"pv_none_match","market":"unknown"}`. A mock-only suite never sees
this line; live smoke against Tencent does. Fixed by dropping the sentinel
symbol in `parseQuoteResponse`, with a regression test
(`TestParseQuoteResponseNoneMatchSentinel`).

## Reproduce

```bash
go build -o cnstock-cli ./cmd/cnstock-cli
./cnstock-cli quote 600519 --format json
./cnstock-cli market --format json
./cnstock-cli sectors --board hy --top 5 --format json
```

> **Needs live verification:** the reverse-engineered Eastmoney endpoints behind
> `financials`, `constituents`, and `moneyflow` (env vars `CNS_FINANCIALS_ENDPOINT`,
> `CNS_CONSTITUENTS_ENDPOINT`, `CNS_MONEYFLOW_ENDPOINT`) are parsed defensively
> against the standard push2 JSON shape but have NOT yet been live-smoked; the
> exact `f`-field IDs and the board-code format for `constituents` must be
> confirmed against the real upstream and adjusted:
>
> ```bash
> ./cnstock-cli financials 600519 --format json
> ./cnstock-cli constituents BK0475 --format json
> ./cnstock-cli moneyflow 600519 --format json
> ```
