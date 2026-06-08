## Summary

What does this PR do? Why?

## Changes

- ...

## Test plan

- [ ] `go test -race ./...` passes
- [ ] `go vet ./...` passes
- [ ] `npm audit --omit=dev --audit-level=high` passes
- [ ] Manual smoke test: `cnstock-cli quote sh600519 --json`

## Checklist

- [ ] Tests added/updated for behavior changes
- [ ] README updated if user-facing flags changed
- [ ] README_zh updated if README changed
- [ ] CHANGELOG.md updated under Unreleased
- [ ] `skills/cnstock-cli/SKILL.md` updated if commands, flags, schema, or error handling changed
- [ ] `cnstock-cli reference`, `context`, `doctor`, and `changelog` still match `.agent/CLI-SPEC.md`
- [ ] External text fields are tagged `_untrusted` and sensitive output is redacted
