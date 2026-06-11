# Open Source Checklist

Use this before a public release or major contract change.

## Repository

- [ ] `README.md` and `README_zh.md` describe the same user-facing behavior.
- [ ] `CHANGELOG.md` has an `Unreleased` entry for the change.
- [ ] `LICENSE`, `NOTICE.md`, `SECURITY.md`, `CONTRIBUTING.md`, and `CODE_OF_CONDUCT.md` are present.
- [ ] `AGENTS.md` points contributors to `.agent/AGENT.md`.
- [ ] `.github/workflows/ci.yml` runs format, lint, vet, tests, and npm audit.
- [ ] `.github/workflows/release.yml` builds traceable release artifacts from tags.

## CLI Contract

- [ ] JSON mode stdout is exactly one envelope.
- [ ] Error envelopes go to stderr and include `ok:false`, `schema_version`, `meta.duration_ms`, `error.code`, `error.details`, and `error.retryable`.
- [ ] `reference`, `context`, `doctor`, and `changelog` are present and current.
- [ ] Exit codes match `reference.data.error_codes`.
- [ ] `--fields` and `--compact` work for JSON output.
- [ ] Functional Contract Coverage is 100%: public README, Skill, `reference`, `--help`, `context`, `doctor`, `changelog`, and `update` behavior has command-level tests.
- [ ] `reference.release_readiness.level` is accurate: `stable` has FCC 100%, mock upstream/contract tests, and recorded live smoke/E2E evidence; missing live evidence is `beta`; missing command-level coverage is `unpublishable`.
- [ ] `doctor` includes a `release_readiness` check whose status matches the declared release level.
- [ ] Market-data read-only commands reject `--dry-run` and `--confirm` instead of silently ignoring them; `update` supports the dry-run/confirm lifecycle.

## Security

- [ ] T0 market-data risk and local-write update boundary are recorded in `SECURITY.md` and `reference`.
- [ ] External text fields are tagged with `_untrusted`.
- [ ] Endpoint URLs are redacted in `context` and `doctor`.
- [ ] npm install verifies checksums and hard-fails if verification is unavailable or mismatched.
- [ ] No credentials, tokens, cookies, or private data are committed.

## Skill

- [ ] `skills/cnstock-cli/SKILL.md` declares `metadata.requires.bins`; add `min_version` when assigning the release version.
- [ ] The Skill points to `cnstock-cli reference` for params, schema, and error codes.
- [ ] The Skill includes activation, non-activation, pre-flight, error handling, `_untrusted`, permission boundary, and 3-6 playbooks.
