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

// TestContractConformance_ErrorCodes asserts every emitted error code is in the
// canonical contract (core ∪ this tool's ext) with the exact exit + retryable.
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
	}
}

func TestContractConformance_SchemaVersion(t *testing.T) {
	if SchemaVersion != contract.SchemaVersion {
		t.Fatalf("schema_version drift: output=%q contract=%q", SchemaVersion, contract.SchemaVersion)
	}
}

// TestContractConformance_EnvelopeKeys asserts the success and error envelopes
// (and meta) carry only the canonical top-level keys.
func TestContractConformance_EnvelopeKeys(t *testing.T) {
	// Capture a success envelope
	successEnv := captureEnvelope(t, func() {
		RenderEnvelope(map[string]any{"x": 1}, nil, false, 0)
	})
	checkEnvelopeKeys(t, successEnv, contract.SuccessEnvelopeKeys, "success")

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
