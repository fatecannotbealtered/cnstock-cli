package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const SchemaVersion = "2.0"

// Envelope is the stable machine-readable response wrapper for JSON output.
type Envelope struct {
	OK            bool             `json:"ok"`
	SchemaVersion string           `json:"schema_version"`
	Data          *json.RawMessage `json:"data,omitempty"`
	Meta          *Meta            `json:"meta,omitempty"`
	Error         *ErrorPayload    `json:"error,omitempty"`
}

// Meta contains non-contractual execution metadata.
type Meta struct {
	DurationMS int64 `json:"duration_ms"`
}

// ErrorPayload describes a machine-readable CLI failure.
type ErrorPayload struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details"`
	Retryable bool           `json:"retryable"`
}

// PrintJSON outputs v as indented JSON to stdout.
func PrintJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json marshal error: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// RenderJSON outputs v as a JSON envelope to stdout.
func RenderJSON(v any, fields []string, compact bool) {
	RenderEnvelope(v, fields, compact, 0)
}

// RenderEnvelope outputs v as a JSON success envelope to stdout, optionally
// filtering data to an ordered set of top-level fields and/or emitting compact
// single-line output.
//
// Field filtering keeps only the requested data keys, in the order given, which keeps
// the output stable and low-token for agent consumption. It applies to a single
// object or to each element of an array of objects.
func RenderEnvelope(v any, fields []string, compact bool, duration time.Duration) {
	data, err := json.Marshal(v)
	if err != nil {
		PrintErrorEnvelope(fmt.Sprintf("json marshal error: %v", err), ErrUnknown, false, nil, compact)
		return
	}
	if len(fields) > 0 {
		data = filterFields(data, fields)
	}

	raw := json.RawMessage(data)
	payload := Envelope{
		OK:            true,
		SchemaVersion: SchemaVersion,
		Data:          &raw,
		Meta:          &Meta{DurationMS: duration.Milliseconds()},
	}
	writeJSON(os.Stdout, payload, compact)
}

// Raw writes s to stdout verbatim (no wrapping, no trailing formatting beyond a newline).
func Raw(s string) {
	fmt.Println(s)
}

// filterFields keeps only the requested keys (in the given order) from a JSON
// object or array-of-objects. On any structural mismatch it returns data unchanged.
func filterFields(data []byte, fields []string) []byte {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return data
	}
	switch trimmed[0] {
	case '[':
		var arr []json.RawMessage
		if err := json.Unmarshal(trimmed, &arr); err != nil {
			return data
		}
		parts := make([]string, 0, len(arr))
		for _, el := range arr {
			parts = append(parts, string(filterObject(el, fields)))
		}
		return []byte("[" + strings.Join(parts, ",") + "]")
	case '{':
		return filterObject(trimmed, fields)
	default:
		return data
	}
}

// filterObject keeps only the requested keys, in order, from a single JSON object.
func filterObject(obj json.RawMessage, fields []string) []byte {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(obj, &m); err != nil {
		return obj
	}
	var b bytes.Buffer
	b.WriteByte('{')
	first := true
	for _, f := range fields {
		raw, ok := m[f]
		if !ok {
			continue
		}
		if !first {
			b.WriteByte(',')
		}
		first = false
		key, _ := json.Marshal(f)
		b.Write(key)
		b.WriteByte(':')
		b.Write(raw)
	}
	b.WriteByte('}')
	return b.Bytes()
}

// ErrorCode classifies errors for machine consumption.
type ErrorCode string

const (
	ErrConfig     ErrorCode = "E_CONFIG"
	ErrAuth       ErrorCode = "E_AUTH"
	ErrForbidden  ErrorCode = "E_FORBIDDEN"
	ErrNotFound   ErrorCode = "E_NOT_FOUND"
	ErrRateLimit  ErrorCode = "E_RATE_LIMITED"
	ErrServer     ErrorCode = "E_SERVER"
	ErrValidation ErrorCode = "E_VALIDATION"
	ErrNetwork    ErrorCode = "E_NETWORK"
	ErrTimeout    ErrorCode = "E_TIMEOUT"
	ErrConfirm    ErrorCode = "E_CONFIRMATION_REQUIRED"
	ErrConflict   ErrorCode = "E_CONFLICT"
	ErrHuman      ErrorCode = "E_HUMAN_REQUIRED"
	ErrUnknown    ErrorCode = "E_UNKNOWN"
)

// PrintErrorJSON outputs an error envelope to stderr.
func PrintErrorJSON(msg string) {
	PrintErrorEnvelope(msg, ErrUnknown, false, nil, false)
}

// PrintErrorJSONWithCode outputs an error envelope with an explicit error code.
func PrintErrorJSONWithCode(msg string, statusCode int, code ErrorCode) {
	details := map[string]any{}
	if statusCode != 0 {
		details["status_code"] = statusCode
	}
	PrintErrorEnvelope(msg, code, isRetryable(code), details, false)
}

// PrintErrorEnvelope outputs a JSON error envelope to stderr.
func PrintErrorEnvelope(msg string, code ErrorCode, retryable bool, details map[string]any, compact bool) {
	PrintErrorEnvelopeWithDuration(msg, code, retryable, details, compact, 0)
}

// PrintErrorEnvelopeWithDuration outputs a JSON error envelope to stderr with
// execution metadata. Error envelopes intentionally mirror success envelopes so
// agents can always inspect ok/schema_version/meta first.
func PrintErrorEnvelopeWithDuration(msg string, code ErrorCode, retryable bool, details map[string]any, compact bool, duration time.Duration) {
	if details == nil {
		details = map[string]any{}
	}
	payload := Envelope{
		OK:            false,
		SchemaVersion: SchemaVersion,
		Meta:          &Meta{DurationMS: duration.Milliseconds()},
		Error: &ErrorPayload{
			Code:      code,
			Message:   msg,
			Details:   details,
			Retryable: retryable,
		},
	}
	writeJSON(os.Stderr, payload, compact)
}

func writeJSON(w io.Writer, v any, compact bool) {
	var (
		data []byte
		err  error
	)
	if compact {
		data, err = json.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", "  ")
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, `{"ok":false,"schema_version":%q,"meta":{"duration_ms":0},"error":{"code":%q,"message":%q,"details":{},"retryable":false}}`+"\n", SchemaVersion, ErrUnknown, err.Error())
		return
	}
	_, _ = fmt.Fprintln(w, string(data))
}

func isRetryable(code ErrorCode) bool {
	switch code {
	case ErrNetwork, ErrServer, ErrRateLimit, ErrTimeout:
		return true
	default:
		return false
	}
}

func hintForCode(code ErrorCode) string {
	switch code {
	case ErrValidation:
		return "Check command arguments and flags"
	case ErrNetwork:
		return "Check network connectivity; for HTTP 5xx, the upstream provider may be unavailable, retry later"
	case ErrServer:
		return "Upstream server returned an error; try again later"
	case ErrNotFound:
		return "Symbol or resource not found; verify the input"
	case ErrConfirm:
		return "Run the command with --dry-run first, then retry with --confirm"
	case ErrConflict:
		return "Refresh state and retry from a new dry-run"
	case ErrHuman:
		return "Complete the requested human action, then resume"
	default:
		return ""
	}
}
