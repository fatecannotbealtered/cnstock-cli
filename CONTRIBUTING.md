# Contributing

Thank you for improving cnstock-cli. This document describes how to build, test, and submit changes.

## Development setup

- Go **1.25+** (see `go.mod`)
- Node.js **16+** if you work on npm packaging or run supply-chain checks
- Optional: **golangci-lint** (CI runs it on Linux)

Before changing command behavior, output contracts, security boundaries, release flow, or the bundled Skill, read [AGENTS.md](AGENTS.md) and the matching spec under [.agent/](.agent/).

Clone and verify:

```bash
git clone https://github.com/fatecannotbealtered/cnstock-cli.git
cd cnstock-cli
go mod download
go test ./...
npm audit --omit=dev --audit-level=high
go build -o bin/cnstock-cli ./cmd/cnstock-cli
./bin/cnstock-cli --help
```

## Commands

| Goal | Command |
|------|---------|
| Run tests (race) | `go test -race ./...` |
| Format | `gofmt -w .` |
| Vet | `go vet ./...` |
| npm audit | `npm audit --omit=dev --audit-level=high` |
| Lint | `golangci-lint run ./...` (or `make lint` on Unix) |
| Build with version | `make build` (Unix) or `go build -ldflags "-s -w -X github.com/fatecannotbealtered/cnstock-cli/cmd.version=dev" -o bin/cnstock-cli.exe ./cmd/cnstock-cli` (Windows) |

## Functional contract coverage

Release standard: **Functional Contract Coverage = 100%**. Every public behavior documented in README, Skill, `cnstock-cli reference`, `--help`, `context`, `doctor`, `changelog`, or `update` must have automated command-level tests.

For each new or changed command, cover success, invalid arguments, config/auth/permission failure where applicable, upstream failure or timeout where applicable, JSON envelope shape, output schema, exit code, stdout/stderr boundary, and non-interactive behavior. Every bug fix that changes observable behavior needs a regression test.

Numeric line coverage is tracked separately and may ratchet upward, but it does not replace missing contract tests.

Release readiness is machine-readable:

- `stable`: FCC is 100%, mock upstream/contract tests cover success and failure paths, and live smoke/E2E evidence is recorded for the release candidate.
- `beta`: FCC is 100% and mock upstream/contract tests are complete, but live smoke/E2E evidence is missing or explicitly unavailable.
- `unpublishable`: any public behavior lacks command-level tests, or mock upstream/contract tests cover only happy paths.

Keep `cnstock-cli reference` `release_readiness` and `cnstock-cli doctor`'s `release_readiness` check honest when test evidence changes.

## Pull requests

1. **One logical change per PR** when possible.
2. **Tests**: add or update tests for behavior changes in `internal/` or stable CLI contracts.
3. **Docs**: update `README.md` and `README_zh.md` together if user-facing flags or flows change; add a line to `CHANGELOG.md` under *Unreleased*.
4. **Skill**: update `skills/cnstock-cli/SKILL.md` when commands, flags, schema, error handling, or minimum version changes.
5. **Contract**: verify `cnstock-cli reference`, `context`, `doctor`, and `changelog` still match `.agent/CLI-SPEC.md`.
6. **Commits**: clear messages; no secrets in code or docs.

## Security

Do not open public issues for undisclosed security vulnerabilities. See [SECURITY.md](SECURITY.md).
