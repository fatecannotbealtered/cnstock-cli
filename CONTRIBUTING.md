# Contributing

Thank you for improving cnstock-cli. This document describes how to build, test, and submit changes.

## Development setup

- Go **1.23+** (see `go.mod`)
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

## Pull requests

1. **One logical change per PR** when possible.
2. **Tests**: add or update tests for behavior changes in `internal/` or stable CLI contracts.
3. **Docs**: update `README.md` and `README_zh.md` together if user-facing flags or flows change; add a line to `CHANGELOG.md` under *Unreleased*.
4. **Skill**: update `skills/cnstock-cli/SKILL.md` when commands, flags, schema, error handling, or minimum version changes.
5. **Contract**: verify `cnstock-cli reference`, `context`, `doctor`, and `changelog` still match `.agent/CLI-SPEC.md`.
6. **Commits**: clear messages; no secrets in code or docs.

## Security

Do not open public issues for undisclosed security vulnerabilities. See [SECURITY.md](SECURITY.md).
