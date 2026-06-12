package cmd

import "testing"

func TestParseChangelog(t *testing.T) {
	const fixture = `# Changelog

## [Unreleased]

### Added

- Add changelog command.

### Changed

- Bump schema version.

### Security

- Harden checksum verification.

## [1.1.0] - 2026-06-07

### Fixed

- Stabilize market endpoint.
`

	entries := parseChangelog(fixture)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Version != "Unreleased" || entries[0].Changes["added"][0] != "Add changelog command." {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if entries[0].Changes["changed"][0] != "Bump schema version." {
		t.Fatalf("unexpected unreleased changed entry: %+v", entries[0])
	}
	if entries[1].Version != "1.1.0" || entries[1].Date != "2026-06-07" {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}

	filtered := filterChangelogSince(entries, "1.0.3")
	if len(filtered) != 2 {
		t.Fatalf("filtered len = %d, want Unreleased + 1.1.0", len(filtered))
	}
	if filtered[1].Version != "1.1.0" {
		t.Fatalf("filtered version = %s, want 1.1.0", filtered[1].Version)
	}
}

func TestBuildReferenceSpecContract(t *testing.T) {
	ref := buildReference()
	if ref.Tool != "cnstock-cli" {
		t.Fatalf("tool = %q, want cnstock-cli", ref.Tool)
	}
	if ref.SchemaVersion != "1.0" {
		t.Fatalf("schema_version = %q, want 2.0", ref.SchemaVersion)
	}
	if ref.RiskTier != riskTier || len(ref.Permissions) == 0 || ref.Permissions[0].Writable {
		t.Fatalf("reference should declare read-only risk boundary: %+v", ref)
	}
	if ref.ReleaseReadiness.Level != "beta" {
		t.Fatalf("release level = %q, want beta", ref.ReleaseReadiness.Level)
	}
	if ref.ReleaseReadiness.LiveSmokeStatus != "missing" {
		t.Fatalf("live smoke status = %q, want missing", ref.ReleaseReadiness.LiveSmokeStatus)
	}

	hasChangelog := false
	for _, c := range ref.Commands {
		if c.Path == "changelog" && c.Type == "self-description" {
			hasChangelog = true
		}
		if c.Mutates {
			t.Fatalf("cnstock-cli command should not mutate external state: %+v", c)
		}
	}
	if !hasChangelog {
		t.Fatal("reference should include changelog command")
	}

	hasValidation := false
	for _, e := range ref.ErrorCodes {
		if e.Code == "E_VALIDATION" && e.ExitCode == ExitBadArgs && !e.Retryable {
			hasValidation = true
		}
	}
	if !hasValidation {
		t.Fatal("reference should include E_VALIDATION exit-code mapping")
	}
}

func TestReleaseReadinessDoctorContract(t *testing.T) {
	readiness := buildReleaseReadiness()
	if readiness.Level != "beta" || readiness.LiveSmokeStatus != "missing" {
		t.Fatalf("unexpected readiness: %+v", readiness)
	}
	if releaseReadinessCheckStatus() != "warn" {
		t.Fatalf("release readiness doctor status = %q, want warn", releaseReadinessCheckStatus())
	}
	if releaseReadinessCheckFix() == "" {
		t.Fatal("beta release readiness should include a fix")
	}
}
