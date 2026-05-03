# Contributing

Thank you for improving cnstock-cli. This document describes how to build, test, and submit changes.

## Development setup

- Go **1.23+** (see `go.mod`)
- Optional: **Node.js 16+** if you work on npm install scripts
- Optional: **golangci-lint** (CI runs it on Linux)

Clone and verify:

```bash
git clone https://github.com/fatecannotbealtered/cnstock-cli.git
cd cnstock-cli
go mod download
go test ./...
go build -o bin/cnstock-cli ./cmd/cnstock-cli
./bin/cnstock-cli --help
```

## Commands

| Goal | Command |
|------|---------|
| Run tests (race) | `go test -race ./...` |
| Format | `gofmt -w .` |
| Vet | `go vet ./...` |
| Lint | `golangci-lint run ./...` (or `make lint` on Unix) |
| Build with version | `make build` (Unix) or `go build -ldflags "-s -w -X github.com/fatecannotbealtered/cnstock-cli/cmd.version=dev" -o bin/cnstock-cli.exe ./cmd/cnstock-cli` (Windows) |

## Pull requests

1. **One logical change per PR** when possible.
2. **Tests**: add or update tests for behavior changes in `internal/` or stable CLI contracts.
3. **Docs**: update `README.md` / `README_zh.md` if user-facing flags or flows change; add a line to `CHANGELOG.md` under *Unreleased*.
4. **Commits**: clear messages; no secrets in code or docs.

## Security

Do not open public issues for undisclosed security vulnerabilities. See [SECURITY.md](SECURITY.md).
