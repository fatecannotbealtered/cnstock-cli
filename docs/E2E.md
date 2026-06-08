# E2E Testing

The E2E suite builds the CLI binary and runs it as a subprocess against local `httptest` servers. This keeps CI deterministic and avoids depending on live Tencent or Eastmoney endpoints.

## Local Commands

```bash
go test ./...
go test ./test/e2e -run TestBinary
```

Race tests need cgo and a C compiler:

```bash
CGO_ENABLED=1 go test -race ./...
```

On Windows, install a GCC-compatible toolchain before running the race detector.

## Endpoint Overrides

All endpoint families support environment-variable overrides for tests and reproductions:

```bash
CNS_QUOTE_ENDPOINT=http://127.0.0.1:8080/q=%s cnstock-cli quote sh600519
CNS_KLINE_ENDPOINT=http://127.0.0.1:8080/appstock/app/%s/get?param=%s cnstock-cli kline sh600519
```

`context` and `doctor` redact credentials and sensitive URL parameters before printing endpoint configuration.

## Live Smoke Checks

Live checks are optional because upstream endpoints are unofficial and may be unavailable or rate-limited:

```bash
cnstock-cli context --compact
cnstock-cli doctor
cnstock-cli reference --compact --fields tool,version,risk_tier,commands
cnstock-cli changelog --since 1.1.0 --compact
```

Do not use live checks as the only evidence for parser correctness; keep fixtures and binary-level tests updated.
