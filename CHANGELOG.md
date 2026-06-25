# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Fleet contract single-source: `contract/contract.json` (vendored from ai-native-cli-spec@v1.4) and the generated `internal/contract/contract_gen.go` are now the canonical exit-code and retryability source for this tool. All error-code→exit and code→retryable lookups in `internal/output` now delegate to `contract.ExitFor`/`contract.Retryable` so the mapping cannot drift. `contract/contract-ext.json` declares the tool-specific `E_HUMAN_REQUIRED` (exit 9) extension code.
- Conformance test `internal/output/contract_conformance_test.go` asserts every emitted `E_*` code is in the canonical contract with exact exit+retryable, schema_version matches, and envelope keys are within the canonical sets — CI-red guard against drift.
- CI step "Verify spec/contract sync" (`node scripts/check-spec.js`) added after "Verify version sync" in the `npm-audit` job, enforcing that vendored `.agent` specs and generated `contract_gen.go` cannot silently fork the template.

### Changed

- `.agent` directory is now single-sourced from ai-native-cli-spec@v1.4 (synced via `node scripts/sync-spec.js`); `.agent/SPEC_VERSION` pins the tag.
- `update` now **drives the package manager** for npm and go-install installs (`npm install -g <pkg>@<ver>` or `go install <pkg>@<ver>`) instead of always running in-process Sigstore+binary-replace regardless of install method. Package-manager integrity is the PM's own; `signature_status` stays `"not_checked"` on this path. A testable seam (`updateRunPackageManager`) allows tests to stub the PM without shelling out. `--dry-run` previews the PM command without executing it; a PM failure reports `E_IO` with `binary_replaced:false` and the manual command.

## [1.1.9] - 2026-06-25

### Changed

- Windows binary self-update now performs the **same in-process rename trick** as Unix: the running executable is moved aside to `.<name>.old`, the freshly verified binary is renamed into place, and on failure the original is rolled back. This replaces the previous `.cmd` replace-on-restart helper script, so `update` completes the swap in one call and reports `status: "updated"` / `binary_replaced: true` immediately — there is no longer a `"scheduled"` status or a restart-deferred pending state.
- `update` now reports a **real `install_method`** (`npm`, `go-install`, or `github-binary`) probed from the running executable's location (node_modules manifest / Go bin dir), instead of a hardcoded `github-binary`. The notice cache surfaces the same detected value.

### Removed

- The dead `update --method` flag (an `auto|npm|go|github` "hint" that nothing read — `install_method` was hardcoded) is removed.

### Fixed

- `update` no longer misclassifies a **failure to download the Sigstore signature bundle** as a non-retryable `E_INTEGRITY` supply-chain failure. The bundle fetch is a network step: a transient fetch failure (or a SIGINT) is now classified by the normal taxonomy (retryable `E_NETWORK`/`E_TIMEOUT`/`E_SERVER`, or `E_INTERRUPTED`) at `stage: "download"`; only an actually invalid/missing-then-refused signature or a checksum mismatch yields `E_INTEGRITY` (CLI-SPEC §14, SEC-SPEC §5).
- `update` **discover-stage** HTTP failures are now classified by status onto the taxonomy instead of collapsing into `E_NETWORK`: `404 → E_NOT_FOUND` (exit 3, non-retryable), `408 → E_TIMEOUT`, `429 → E_RATE_LIMITED`, `5xx → E_SERVER` (all retryable). The status→code→exit mapping now lives in a single shared `api.ErrorForStatus`, used by both the data client and the self-update path so it cannot drift (CLI-SPEC §6).
- `update`'s **latest-release probe** (`fetchLatestRelease` — the first network call of every update and the notice-refresh path) is now routed through the same `api.ErrorForStatus` mapping as the binary-release discover: a real `404` on `/releases/latest` (the release or repo is gone) is the non-retryable `E_NOT_FOUND` (exit 3) the contract requires, rather than a retryable `E_SERVER`. Previously only the binary-release discover had been migrated, so this main-path probe still collapsed every non-`200` (except `5xx`) into a retryable server error (CLI-SPEC §6).
- The `code → retryable` decision in the staged update-failure envelope is now derived from the single `output.IsRetryable` predicate instead of a duplicated local table, so the two cannot disagree.
- `install_method` detection now resolves symlinks on each candidate Go bin dir before comparing it to the (already symlink-resolved) executable path, so a `go-install` layout behind a symlinked directory — macOS `/var → /private/var`, a Windows short path — is no longer misreported as `github-binary`.

## [1.1.8] - 2026-06-22

### Added

- The update-available notice now also rides along on **every** command's `meta.notices` (read-only from the local update-check cache — no network I/O, no phone-home; omitted when the cache holds nothing). The fresh/active view stays on `data.notices` for `context`/`doctor`/`update --check`; `meta.notices` is the cached view available on all commands.
- Update notices are now **severity-graded** (CLI-SPEC §14): the notice carries `warning` when the embedded CHANGELOG delta since the running version contains a `security` entry, or when the latest release crosses a major version; otherwise `info`. Severity is computed at check time and stored in the cache, so the cached `meta.notices` carries the right level. (`critical` is reserved and not emitted from the changelog delta.)

## [1.1.7] - 2026-06-21

### Changed

- `update` is now a **single command with no confirm token**. A bare `cnstock-cli update` performs the whole self-update in one call — resolve the latest (or `--target-version`) release, verify integrity in-process (Sigstore signature then SHA256 checksum), replace the binary, then sync the whole Agent Skill directory. The previous `--dry-run` → `--confirm <token>` write gate is removed from `update` (self-update is exempt from the §7 write gate; its safety guarantee is in-process signature verification, not an agent's review of a preview). `update` is idempotent: already-latest returns `ok` with a no-op. `--check` and `--dry-run` remain optional read-only flags, and `--dry-run` no longer issues a `confirm_token` or `expires_at`. The `--confirm` global flag and the update confirm-token machinery are removed.

### Added

- Staged failure and interruption envelope for `update` (CLI-SPEC §14): every update failure and interrupt now carries `stage` (`discover`/`download`/`verify_signature`/`verify_checksum`/`replace`/`skill_sync`), `current_version`, `binary_replaced`, and `skill_sync_status` so an agent can reason about the post-state. A SIGINT/SIGTERM during `update` is trapped, the temp dir is cleaned, and a terminal JSON envelope (`E_INTERRUPTED`, exit 130) is still emitted instead of dying as a bare killed process.
- New error codes `E_IO` (→ exit 1) and `E_INTERRUPTED` (→ exit 130).

### Fixed

- `update` replace-stage local failures (temp dir, extract, file write/rename, permission, disk) are no longer misclassified as a retryable `E_NETWORK`: permission failures map to `E_FORBIDDEN` (exit 4) and other io/disk failures to `E_IO` (exit 1), both non-retryable with `binary_replaced: false`.
- A Skill-sync failure after a successful binary replace is now reported as **partial success** (`ok: false`, `binary_replaced: true`, retryable) with a `skill_sync_command`, instead of a hard `E_NETWORK` that hid the fact the binary already updated.

## [1.1.6] - 2026-06-16

### Fixed

- npm `optionalDependencies` platform-package pins now match the package version. The previous release bumped the top-level version but left the pins at the prior version, so `npm install` resolved a stale platform binary (the new wrapper with the old binary). The publish workflow now rewrites `optionalDependencies` from the package version before `npm publish`, so the pins can no longer drift from the single source of truth.

## [1.1.5] - 2026-06-16

### Changed

- `update` is rewritten from package-manager delegation (npm/go install) to a self-contained verified binary self-update: download the release archive + `checksums.txt` + Sigstore bundle, verify the signature **in-process** (embedded `sigstore-go`, embedded TUF root) against this repo's tagged release-workflow identity, verify the archive SHA256, and replace the running binary — no dependency on npm/go/pip being installed. Releases are signed with `cosign sign-blob --new-bundle-format`.

### Security

- Verification is mandatory and fail-closed (no skip path); release-integrity failures return the non-retryable `E_INTEGRITY` code (exit 1) instead of a retryable network code.

## [1.1.4] - 2026-06-15

### Added

- **Batch market-data queries** following a single batch contract (plural input, per-item aggregated `items[]` + `summary`, `--continue-on-error`):
  - `financials <symbols>` — comma-separated batch served natively by the multi-code Tencent quote endpoint (class A); each symbol is aggregated per item, a missing symbol surfaces as a per-item `E_NOT_FOUND` rather than a whole-command failure.
  - `kline <symbols>` — comma-separated batch; the single-code upstream is looped client-side (class B) and aggregated into the same shape, indistinguishable from a native batch.
  - `--continue-on-error` (default `true`): best-effort finishes the whole batch; set `false` to stop at the first failure (succeeded items are kept, remaining symbols reported as `summary.skipped`).
  - A command-wide argument error (bad `--limit`/`--adj`/`--from`/`--to`) fails the whole batch with top-level `E_VALIDATION`, not a per-item error.

### Changed

- `kline`, `financials`, and `minute` now take the plural `--symbols` input convention for cross-command consistency with `quote`; a single value degrades to a batch of one. `minute` adopts the plural input but rejects more than one symbol with `E_VALIDATION` (multi-symbol intraday fetch is deferred until the upstream's multi-code support is confirmed).
- `reference` now exposes `kline_batch` / `financials_batch` output schemas (the `items[]`/`summary` shape, with `items[].data.name` listed under `untrusted_fields`) plus batch `examples[]`.
- npm scope 迁移 `@fatecannotbealtered-` → `@fateforge`（无横线 org 在 npm 被占，迁移到 `@fateforge`）。

## [1.1.3] - 2026-06-14

### Added

- `financials <symbol>` — company fundamentals (total/circulating market cap, PE, PB, turnover rate, amount), sourced from the reliable Tencent quote endpoint.
- `kline --from/--to` date-range window; `market --date YYYYMMDD` for a deterministic limit-up/down pool.
- `reference` now exposes a real per-command `output_schema` (label → fields/untrusted catalog) and a runnable `examples[]`, guarded against regression.

### Changed

- Confirm tokens are now single-use: replaying a confirmed `update` token returns `E_CONFLICT` (safe-retry).

## [1.1.2] - 2026-06-14

### Added

- Upstream HTTP client-error statuses now map onto the error taxonomy so an agent can distinguish failure modes via `error.code` + `retryable`: 404 → `E_NOT_FOUND`, 429 → `E_RATE_LIMITED` (retryable), 401 → `E_AUTH`, 403 → `E_FORBIDDEN`, 5xx → `E_SERVER` (retryable). Previously every 4xx collapsed to `E_NETWORK`.

### Fixed

- Corrected the `reference` output-contract and code docstrings to match the actual (and spec-mandated) behavior: the JSON failure envelope is emitted on **stdout** (the single document agents parse), not stderr. No runtime behavior changed — only the self-description was wrong.

## [1.1.1] - 2026-06-13

### Added

- Recorded live smoke against the real Eastmoney/Tencent upstreams (`docs/LIVE-SMOKE-EVIDENCE.md`, 2026-06-13: all 11 leaf commands PASS); `release_readiness` is now `stable` with `live_smoke_status: verified`.
- FCC enumeration guard (`TestFCC_EveryLeafCommandHasTest`): enumerates every leaf command from live `reference` output and asserts each has a command-level test; skips while `fcc_status` is honestly declared non-verified, so the claim cannot be flipped without coverage.
- Command-level e2e tests for `sectors` (mock rank endpoint, `_untrusted` tagging, E_VALIDATION path), `market` (mock breadth/limit pools, upstream-500 path), and `doctor` (all probe targets mocked, release_readiness check asserted) — the three leaves the guard found uncovered.
- **`changelog` command**: Emits version changes derived from `CHANGELOG.md`, with `--since <version>` for agent knowledge refresh after updates.
- **Agent-native repository entrypoints**: Added `AGENTS.md`, `NOTICE.md`, `CODE_OF_CONDUCT.md`, compatibility notes, E2E notes, and an open-source checklist aligned with `.agent/` specs.
- **Security self-description**: `reference`, `context`, and `doctor` now declare the T0 market-data boundary, local-write update boundary, credential status, and version readiness.
- **Lifecycle update flow**: `update` now supports `--check`, `--dry-run`, and `--confirm <confirm_token>` for local package/binary updates plus Agent Skill sync.

### Changed

- Synced `.agent/` SEC-SPEC from the template: credential-at-rest is now the keyring three-part pattern (password discarded after login / secrets in the OS keyring / zero-secret config), file encryption demoted to a visible fallback, env vars as the recommended secret channel, and an honest note on Windows `0600` semantics.
- In JSON mode the failure envelope is now the single JSON document on stdout, matching CLI-SPEC §4; stderr stays a human-readable side channel.
- Synced the `.agent/` spec copies from the ai-native-cli-spec template: stdout failure envelope (§4), HMAC confirm-token requirement (§7), signature_status/signature_verified fields (§14), Skill frontmatter `version` rule.
- Unified the golangci-lint v2 toolchain: Makefile installs from the `/v2` module path and CI uses `golangci-lint-action@v8` to match the v2 config format.
- **Schema version stays 1.0**: the machine contract was reshaped pre-release; with no published consumers, the first public contract starts at `1.0`.
- **Error taxonomy**: Validation failures now use `E_VALIDATION` instead of `E_BAD_ARGS`; error envelopes include `meta.duration_ms`.
- **Reference contract**: `reference` now reports command params, output schemas, permission tier, risk tier, global write-confirmation flags, and `_untrusted` fields as the machine truth source.
- **Local lifecycle writes**: market-data commands still reject `--dry-run` and `--confirm`; `update` uses them to preview and confirm self-update plus whole Skill directory sync through `npx skills add fatecannotbealtered/cnstock-cli -y -g`.
- **Time fields**: Quote and minute JSON time fields now emit UTC ISO 8601 strings.
- **Skill alignment**: The bundled Skill now points to `cnstock-cli reference` for drift-prone params, schemas, error codes, and update lifecycle rules.

### Security

- Confirm tokens are now signed with a machine-local HMAC key (`confirm.secret`, created on first use with 0600 permissions) so they cannot be fabricated without running `--dry-run` on the same machine.
- **Untrusted external content**: Externally sourced text fields now include `_untrusted` markers so agents treat them as data, not instructions.
- **Endpoint redaction**: `context` and `doctor` redact URL credentials and sensitive query parameters before printing endpoint configuration.
- **npm install integrity**: `scripts/install.js` now hard-fails when checksum verification is unavailable or the archive is missing from `checksums.txt`.
- **Signed release checksums**: Release checksums are signed with Sigstore/Cosign, and install/update paths report signature verification status separately from checksum verification.
- **Supply-chain gate**: CI and release workflows now run `npm audit --omit=dev --audit-level=high`.

### Fixed

- `quote` surfaced a phantom `{"symbol":"pv_none_match"}` row when a query matched nothing: Tencent's `v_pv_none_match` sentinel rides in the symbol field, but the parser guard only checked the data field. Now dropped, with a regression test. Found by live smoke against the real Tencent endpoint.


## [1.1.0] - 2026-06-07

### Added

- **`update` command**: Checks the latest GitHub Release and prints safe update instructions for npm, Go install, or direct GitHub binary downloads. The command is read-only and does not replace the running binary.

### Changed

- **Agent JSON envelope**: Default JSON output now returns a stable envelope with `ok`, `schema_version`, `data`, and `meta.duration_ms`. `--fields` filters inside `data` while preserving the envelope.
- **Error envelope and codes**: JSON errors now use `ok:false` with `error.code`, `error.message`, `error.details`, and `error.retryable`. Error codes now follow the `E_*` contract (`E_BAD_ARGS`, `E_NOT_FOUND`, `E_NETWORK`, `E_TIMEOUT`, etc.).
- **Exit codes aligned for agents**: Exit code `3` is now resource-not-found, `4` is auth/permission, `7` is retryable transient failure, and `8` is timeout.
- **`reference` output**: `cnstock-cli reference` now emits machine-readable JSON by default. Use `--format text` for the Markdown view.

### Fixed

- **`market` command stability**: Switch the Eastmoney breadth endpoint to the webguest `ulist.np` path and use host-appropriate Referer headers for Eastmoney requests.

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
- **npm distribution**: `npm install -g @fateforge/cnstock-cli` with bundled AI Agent Skill.
- **Bilingual README**: English + Chinese.
- **Test suite**: 48 test cases across 3 layers — CLI smoke tests (6), API-level integration with httptest mock (28), and binary-level E2E with exec + httptest + env var endpoint injection (14).

### Documentation

- SKILL.md with JSON output schemas, error codes, and exit codes.
- Reference command for AI self-discovery.
- **Data source disclaimer**: Endpoints are from Tencent Finance public web pages, NOT an official API. No SLA, no schema contract, no rate-limit policy. For personal/research use only.
- **Non-affiliation**: Not an official Tencent product; data rights remain with their respective holders.

[Unreleased]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.0.3...v1.1.0
[1.0.3]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/fatecannotbealtered/cnstock-cli/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/fatecannotbealtered/cnstock-cli/releases/tag/v1.0.0
