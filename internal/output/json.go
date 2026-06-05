package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PrintJSON outputs v as indented JSON to stdout.
func PrintJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json marshal error: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// RenderJSON outputs v as JSON to stdout, optionally filtering to an ordered set
// of top-level fields and/or emitting compact single-line output.
//
// Field filtering keeps only the requested keys, in the order given, which keeps
// the output stable and low-token for agent consumption. It applies to a single
// object or to each element of an array of objects.
func RenderJSON(v any, fields []string, compact bool) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json marshal error: %v\n", err)
		return
	}
	if len(fields) > 0 {
		data = filterFields(data, fields)
	}
	if !compact {
		var buf bytes.Buffer
		if err := json.Indent(&buf, data, "", "  "); err == nil {
			data = buf.Bytes()
		}
	}
	fmt.Println(string(data))
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
	ErrConfig     ErrorCode = "CONFIG_ERROR"
	ErrAuth       ErrorCode = "AUTH_REQUIRED"
	ErrForbidden  ErrorCode = "FORBIDDEN"
	ErrNotFound   ErrorCode = "NOT_FOUND"
	ErrRateLimit  ErrorCode = "RATE_LIMITED"
	ErrServer     ErrorCode = "SERVER_ERROR"
	ErrValidation ErrorCode = "VALIDATION_ERROR"
	ErrNetwork    ErrorCode = "NETWORK_ERROR"
	ErrUnknown    ErrorCode = "UNKNOWN_ERROR"
)

// PrintErrorJSON outputs an error message as JSON to stderr.
func PrintErrorJSON(msg string) {
	payload := struct {
		Error     string    `json:"error"`
		ErrorCode ErrorCode `json:"errorCode"`
	}{
		Error:     msg,
		ErrorCode: ErrUnknown,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error": %q, "errorCode": %q}`+"\n", msg, ErrUnknown)
		return
	}
	fmt.Fprintln(os.Stderr, string(data))
}

// PrintErrorJSONWithCode outputs an error with an explicit error code.
func PrintErrorJSONWithCode(msg string, statusCode int, code ErrorCode) {
	payload := struct {
		Error      string    `json:"error"`
		StatusCode int       `json:"statusCode,omitempty"`
		ErrorCode  ErrorCode `json:"errorCode"`
		Hint       string    `json:"hint,omitempty"`
	}{
		Error:      msg,
		StatusCode: statusCode,
		ErrorCode:  code,
		Hint:       hintForCode(code),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error": %q, "errorCode": %q}`+"\n", msg, code)
		return
	}
	fmt.Fprintln(os.Stderr, string(data))
}

func hintForCode(code ErrorCode) string {
	switch code {
	case ErrValidation:
		return "Check command arguments and flags"
	case ErrNetwork:
		return "Check network connectivity and try again"
	case ErrServer:
		return "Upstream server returned an error; try again later"
	case ErrNotFound:
		return "Symbol or resource not found; verify the input"
	default:
		return ""
	}
}
