package cmd

import (
	"testing"

	"github.com/fatecannotbealtered/cnstock-cli/internal/output"
)

// TestReference_NewErrorCodeExitMapping pins the status->code->exit contract for
// the update error codes added in CLI-SPEC §14: E_IO -> 1, E_INTERRUPTED -> 130,
// E_INTEGRITY -> 1. The reference table is the single machine-readable surface an
// agent reads, so it must declare these mappings.
func TestReference_NewErrorCodeExitMapping(t *testing.T) {
	want := map[output.ErrorCode]struct {
		exit      int
		retryable bool
	}{
		output.ErrIO:          {ExitIO, false},
		output.ErrInterrupted: {ExitInterrupted, true},
		output.ErrIntegrity:   {ExitGeneric, false},
	}
	if ExitIO != 1 || ExitInterrupted != 130 {
		t.Fatalf("exit constants drifted: ExitIO=%d ExitInterrupted=%d, want 1 and 130", ExitIO, ExitInterrupted)
	}

	ref := buildReference()
	seen := map[output.ErrorCode]bool{}
	for _, ec := range ref.ErrorCodes {
		exp, ok := want[ec.Code]
		if !ok {
			continue
		}
		seen[ec.Code] = true
		if ec.ExitCode != exp.exit {
			t.Errorf("%s exit = %d, want %d", ec.Code, ec.ExitCode, exp.exit)
		}
		if ec.Retryable != exp.retryable {
			t.Errorf("%s retryable = %v, want %v", ec.Code, ec.Retryable, exp.retryable)
		}
	}
	for code := range want {
		if !seen[code] {
			t.Errorf("reference error-codes table is missing %s", code)
		}
	}
}
