package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/fatecannotbealtered/cnstock-cli/internal/contract"
)

// allErrorCodes enumerates every ErrorCode this tool can emit. Keep in sync with
// the const block in json.go; the conformance test asserts each is part of the
// canonical fleet contract (contract/contract.json, single-sourced from the
// ai-native-cli-spec template) with the exact exit code and retryability.
var allErrorCodes = []ErrorCode{
	ErrConfig, ErrAuth, ErrForbidden, ErrNotFound, ErrRateLimit, ErrServer,
	ErrValidation, ErrNetwork, ErrTimeout, ErrConfirm, ErrConflict, ErrHuman,
	ErrIntegrity, ErrIO, ErrInterrupted, ErrUnknown,
}

// independentCodeTable is an independent hardcoded expected mapping of the 16
// canonical E_* codes to their exit code and retryability. It is intentionally
// NOT derived from contract.Codes so it can catch a wrong contract.json.
// Canonical table (CLI-SPEC exit_codes):
//
//	E_USAGE/E_VALIDATION=2 (non-retryable)
//	E_NOT_FOUND=3 (non-retryable)
//	E_AUTH/E_FORBIDDEN/E_CONFIG=4 (non-retryable)
//	E_CONFIRMATION_REQUIRED=5 (non-retryable)
//	E_CONFLICT=6 (non-retryable)
//	E_NETWORK/E_RATE_LIMITED/E_SERVER=7 (retryable)
//	E_TIMEOUT=8 (retryable)
//	E_INTEGRITY/E_IO/E_UNKNOWN=1 (non-retryable)
//	E_INTERRUPTED=130 (retryable)
var independentCodeTable = map[ErrorCode]struct {
	exit      int
	retryable bool
}{
	ErrConfig:      {exit: 4, retryable: false},
	ErrAuth:        {exit: 4, retryable: false},
	ErrForbidden:   {exit: 4, retryable: false},
	ErrNotFound:    {exit: 3, retryable: false},
	ErrRateLimit:   {exit: 7, retryable: true},
	ErrServer:      {exit: 7, retryable: true},
	ErrValidation:  {exit: 2, retryable: false},
	ErrNetwork:     {exit: 7, retryable: true},
	ErrTimeout:     {exit: 8, retryable: true},
	ErrConfirm:     {exit: 5, retryable: false},
	ErrConflict:    {exit: 6, retryable: false},
	ErrHuman:       {exit: 9, retryable: false},
	ErrIntegrity:   {exit: 1, retryable: false},
	ErrIO:          {exit: 1, retryable: false},
	ErrInterrupted: {exit: 130, retryable: true},
	ErrUnknown:     {exit: 1, retryable: false},
}

// TestContractConformance_ErrorCodes asserts every emitted error code is in the
// canonical contract (core ∪ this tool's ext) with the exact exit + retryable,
// and also validates against the independent hardcoded table (3c).
func TestContractConformance_ErrorCodes(t *testing.T) {
	for _, c := range allErrorCodes {
		spec, ok := contract.Codes[string(c)]
		if !ok {
			t.Errorf("error code %q is not in the canonical contract (core∪ext)", c)
			continue
		}
		if got := ExitCodeForErrorCode(c); got != spec.Exit {
			t.Errorf("exit drift for %q: tool=%d contract=%d", c, got, spec.Exit)
		}
		if got := IsRetryable(c); got != spec.Retryable {
			t.Errorf("retryable drift for %q: tool=%v contract=%v", c, got, spec.Retryable)
		}

		// Independent check (3c): assert against the hardcoded expected table so a
		// wrong contract.json cannot silently pass by making the two sides agree.
		expected, hasExpected := independentCodeTable[c]
		if !hasExpected {
			t.Errorf("code %q missing from the independent expected table", c)
			continue
		}
		if got := ExitCodeForErrorCode(c); got != expected.exit {
			t.Errorf("independent exit drift for %q: got=%d want=%d", c, got, expected.exit)
		}
		if got := IsRetryable(c); got != expected.retryable {
			t.Errorf("independent retryable drift for %q: got=%v want=%v", c, got, expected.retryable)
		}
	}
}

func TestContractConformance_SchemaVersion(t *testing.T) {
	if SchemaVersion != contract.SchemaVersion {
		t.Fatalf("schema_version drift: output=%q contract=%q", SchemaVersion, contract.SchemaVersion)
	}
}

// TestContractConformance_EnvelopeKeys asserts the success and error envelopes
// (and meta) carry only the canonical top-level keys, that required keys are
// present, that meta.duration_ms (and all MetaRequiredKeys) are present (3a),
// and that success envelopes carry a non-empty "data" key (3b).
func TestContractConformance_EnvelopeKeys(t *testing.T) {
	// Capture a success envelope with a non-empty data payload (3b).
	successEnv := captureEnvelope(t, func() {
		RenderEnvelope(map[string]any{"x": 1}, nil, false, 0)
	})
	checkEnvelopeKeys(t, successEnv, contract.SuccessEnvelopeKeys, "success")

	// (3b) success envelope must carry a "data" key with a non-empty payload.
	dataRaw, hasData := successEnv["data"]
	if !hasData {
		t.Error("success envelope missing required key \"data\"")
	} else if len(dataRaw) == 0 || string(dataRaw) == "null" {
		t.Errorf("success envelope \"data\" is empty/null; must carry a non-empty payload")
	}

	// Capture an error envelope
	errorEnv := captureEnvelope(t, func() {
		PrintErrorEnvelopeWithDuration("test error", ErrValidation, false, nil, false, 0)
	})
	checkEnvelopeKeys(t, errorEnv, contract.ErrorEnvelopeKeys, "error")
}

func captureEnvelope(t *testing.T, fn func()) map[string]json.RawMessage {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = origStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	var top map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &top); err != nil {
		t.Fatalf("envelope is not valid JSON: %v\nraw: %s", err, buf.String())
	}
	return top
}

func checkEnvelopeKeys(t *testing.T, top map[string]json.RawMessage, canonical []string, label string) {
	t.Helper()
	// Flag only UNEXPECTED keys ("data"/"error" are omitempty and may be absent).
	for k := range top {
		if !containsStr(canonical, k) && k != "data" && k != "error" {
			t.Errorf("%s envelope has unexpected top-level key %q (canonical: %v)", label, k, canonical)
		}
	}
	for _, req := range []string{"ok", "schema_version", "meta"} {
		if _, ok := top[req]; !ok {
			t.Errorf("%s envelope missing required key %q", label, req)
		}
	}
	var meta map[string]json.RawMessage
	if raw, ok := top["meta"]; ok {
		_ = json.Unmarshal(raw, &meta)
	}
	// (3a) Assert every MetaRequiredKey is PRESENT in the emitted meta.
	for _, req := range contract.MetaRequiredKeys {
		if _, ok := meta[req]; !ok {
			t.Errorf("%s meta missing required key %q (contract.MetaRequiredKeys=%v)", label, req, contract.MetaRequiredKeys)
		}
	}
	allowed := append(append([]string{}, contract.MetaRequiredKeys...), contract.MetaOptionalKeys...)
	for k := range meta {
		if !containsStr(allowed, k) {
			t.Errorf("meta has unexpected key %q (canonical: %v)", k, allowed)
		}
	}
}

func containsStr(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

// Compile-time check that time.Duration zero is accepted (RenderEnvelope signature).
var _ = time.Duration(0)
