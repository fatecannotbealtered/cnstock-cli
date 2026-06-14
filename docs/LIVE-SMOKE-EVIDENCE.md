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

## `financials` — endpoint correction (2026-06-14)

The reverse-engineered Eastmoney `push2/api/qt/stock/get` endpoint that
originally backed `financials` returns EMPTY/EOF from real networks, so the
command was re-pointed at a source verified working live:

- **`financials` now parses the same Tencent quote string the `quote` command
  uses** (`qt.gtimg.cn`, `CNS_QUOTE_ENDPOINT`). The `~`-delimited line carries
  the valuation tail (index 37=成交额万, 38=换手率, 39=市盈率, 44=流通市值亿,
  45=总市值亿, 46=市净率), cross-referenced against the existing quote parser
  and confirmed against a live response. **Live result: PASS.**

  ```bash
  ./cnstock-cli financials 600519 --format json
  # -> 贵州茅台: market_cap 1.615e12, float_market_cap 1.615e12,
  #    pe_ratio 19.52, pb 6.03, turnover_rate 0.4, amount 6.478e9
  ```

- **`constituents` and `moneyflow` were dropped this round.** Index components
  are not an Eastmoney "board" and no reliable simple public endpoint was found
  (clist `fs=i:`/`fs=b:` returned empty). The `moneyflow` fund-flow kline
  endpoint (`push2.eastmoney.com/api/qt/stock/fflow/kline/get`) is intermittently
  EOF from this environment and could not be honestly live-verified this round.
  Both will return with a proper, verified data source in a future round.

## Batch commands — live smoke (2026-06-15)

The new comma-separated batch surface (`kline` / `financials` aggregated
`items[]`/`summary` envelope, `minute` plural-input single-symbol guard) smoked
against the same public upstreams. All commands here are read-only quote
queries — there is no destructive or irreversible batch in this tool — so every
case below was run live against real endpoints.

- **Environment:** Windows, Go (built from `./cmd/cnstock-cli`); Tencent
  (`qt.gtimg.cn`) for financials, Eastmoney for kline; live public internet.
- **Method:** each invoked with `--format json`; asserted envelope `ok`, the
  `items[]` per-target shape, `summary` tally (`total = succeeded + failed`),
  per-item `{code, retryable}` taxonomy, and exit code.
- **Small batches only:** 2–3 symbols per call (贵州茅台/平安银行/中国平安),
  no writes, nothing to clean up.

| Case | Command | Result | Method |
|---|---|---|---|
| Batch happy-path (kline) | `kline 600519,000001,sh601318 --period day` | PASS (live) | 3 items ok, ordered, normalized targets; summary 3/3/0 |
| Batch happy-path (financials) | `financials 600519,000001,sh601318` | PASS (live) | 3 items ok, names resolved; summary 3/3/0 |
| Partial-failure aggregation | `kline 600519,ZZZ999,000001` | PASS (live) | `ok:true`, summary 3/2/1, bad item carries `E_NOT_FOUND` retryable=false; good items unaffected |
| `--continue-on-error=false` stop+skip | `kline ZZZ999,600519,000001 --continue-on-error=false` | PASS (live) | stops at first failure, summary failed=1 **skipped=2**, only failed item present |
| Dedup + first-seen order | `kline 000001,600519,000001` | PASS (live) | duplicate collapsed → total=2, order `[sz000001, sh600519]` |
| minute multi-symbol reject | `minute 600519,000001` | PASS (live) | `E_VALIDATION` exit 2, honest "not yet available" — no silent first-only fetch |
| Whole-batch arg error | `kline ""` | PASS (live) | top-level `ok:false` `E_VALIDATION` exit 2 |
| minute single still works | `minute 600519` | PASS (live) | 242 ticks |
| reference self-description | `reference` | PASS (live) | `kline_batch` + `financials_batch` schemas present |

**Not tested / not applicable:** no destructive batch (delete / merge / mass
send) exists in this tool, so the "dry-run-only" red line had nothing to gate.
No credentials are involved — all upstreams are anonymous public quote APIs.

Corroborated by `go test ./...` (cmd, internal/api batch aggregation, e2e
binary batch cases, lint) — all green.

### Reproduce

```bash
go build -o cnstock-cli ./cmd/cnstock-cli
./cnstock-cli kline 600519,000001,sh601318 --period day --limit 2 --format json
./cnstock-cli financials 600519,000001,sh601318 --format json
./cnstock-cli kline 600519,ZZZ999,000001 --format json                       # partial failure
./cnstock-cli kline ZZZ999,600519,000001 --continue-on-error=false --format json  # stop+skipped
./cnstock-cli minute 600519,000001 --format json                             # E_VALIDATION
```
