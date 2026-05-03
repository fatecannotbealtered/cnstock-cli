package output

import (
	"encoding/json"
	"fmt"
	"os"
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
