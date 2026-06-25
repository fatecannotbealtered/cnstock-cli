package api

import (
	"errors"
	"fmt"
	"net/http"
)

// ValidationError indicates bad arguments (e.g. invalid limit, empty keyword).
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

// ServerError indicates an upstream business error (e.g. code != 0).
type ServerError struct {
	Msg string
}

func (e *ServerError) Error() string { return e.Msg }

// NetworkError indicates a connection or HTTP transport failure.
type NetworkError struct {
	Msg string
}

func (e *NetworkError) Error() string { return e.Msg }

// NotFoundError indicates the upstream returned no data for the requested symbol.
type NotFoundError struct {
	Msg string
}

func (e *NotFoundError) Error() string { return e.Msg }

// RateLimitError indicates the upstream throttled the request (HTTP 429).
type RateLimitError struct {
	Msg string
}

func (e *RateLimitError) Error() string { return e.Msg }

// AuthError indicates the upstream rejected the request as unauthenticated (HTTP 401).
type AuthError struct {
	Msg string
}

func (e *AuthError) Error() string { return e.Msg }

// ForbiddenError indicates the upstream refused the request (HTTP 403).
type ForbiddenError struct {
	Msg string
}

func (e *ForbiddenError) Error() string { return e.Msg }

// TimeoutError indicates the upstream reported a request timeout (HTTP 408).
type TimeoutError struct {
	Msg string
}

func (e *TimeoutError) Error() string { return e.Msg }

// newValidationError creates a ValidationError.
func newValidationError(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

// NewValidationError creates a ValidationError (exported for use by the cmd layer).
func NewValidationError(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

// newServerError creates a ServerError.
func newServerError(format string, args ...any) error {
	return &ServerError{Msg: fmt.Sprintf(format, args...)}
}

// NewServerError creates a ServerError (exported for use by the cmd layer).
func NewServerError(format string, args ...any) error {
	return &ServerError{Msg: fmt.Sprintf(format, args...)}
}

// newNetworkError creates a NetworkError.
func newNetworkError(format string, args ...any) error {
	return &NetworkError{Msg: fmt.Sprintf(format, args...)}
}

// NewNetworkError creates a NetworkError (exported for use by the cmd layer).
func NewNetworkError(format string, args ...any) error {
	return &NetworkError{Msg: fmt.Sprintf(format, args...)}
}

// newNotFoundError creates a NotFoundError.
func newNotFoundError(format string, args ...any) error {
	return &NotFoundError{Msg: fmt.Sprintf(format, args...)}
}

// statusRetryable reports whether an upstream HTTP status is a transient failure
// worth retrying: 5xx, 429 (rate-limited), and 408 (timeout). Other 4xx are client
// errors and are not retried.
func statusRetryable(statusCode int) bool {
	switch {
	case statusCode >= 500:
		return true
	case statusCode == http.StatusTooManyRequests, statusCode == http.StatusRequestTimeout:
		return true
	default:
		return false
	}
}

// ErrorForStatus maps an upstream HTTP status onto the error taxonomy so callers
// classify failure modes by status (404 vs 429 vs 5xx) instead of collapsing
// every non-2xx into E_NETWORK. This is the single source of the status->error
// mapping (CLI-SPEC §6); the client and the self-update path both route through it
// so the status->code->exit contract cannot drift. A 2xx status returns nil.
func ErrorForStatus(statusCode int, format string, args ...any) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	switch {
	case statusCode == http.StatusNotFound:
		return &NotFoundError{Msg: msg}
	case statusCode == http.StatusRequestTimeout:
		return &TimeoutError{Msg: msg}
	case statusCode == http.StatusTooManyRequests:
		return &RateLimitError{Msg: msg}
	case statusCode == http.StatusUnauthorized:
		return &AuthError{Msg: msg}
	case statusCode == http.StatusForbidden:
		return &ForbiddenError{Msg: msg}
	case statusCode >= 500:
		return &ServerError{Msg: msg}
	default:
		return &NetworkError{Msg: msg}
	}
}

// Quote represents a real-time stock quote.
type Quote struct {
	Symbol    string   `json:"symbol"`
	Market    string   `json:"market"`
	Name      string   `json:"name,omitempty"`
	Code      string   `json:"code,omitempty"`
	Price     *float64 `json:"price,omitempty"`
	PrevClose *float64 `json:"prev_close,omitempty"`
	Open      *float64 `json:"open,omitempty"`
	Volume    *float64 `json:"volume,omitempty"`
	Time      string   `json:"time,omitempty"`
	Change    *float64 `json:"change,omitempty"`
	ChangePct *float64 `json:"change_pct,omitempty"`
	High      *float64 `json:"high,omitempty"`
	Low       *float64 `json:"low,omitempty"`
	Amount    *float64 `json:"amount,omitempty"`
	PeRatio   *float64 `json:"pe_ratio,omitempty"`
	Turnover  *float64 `json:"turnover,omitempty"`
	// A-share specific
	BuyVol  *float64     `json:"buy_vol,omitempty"`
	SellVol *float64     `json:"sell_vol,omitempty"`
	Bid     []DepthLevel `json:"bid,omitempty"`
	Ask     []DepthLevel `json:"ask,omitempty"`
	// HK/US specific
	High52W    *float64 `json:"high_52w,omitempty"`
	Low52W     *float64 `json:"low_52w,omitempty"`
	NameEN     string   `json:"name_en,omitempty"`
	Currency   string   `json:"currency,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	FieldCount int      `json:"field_count,omitempty"`
	Error      string   `json:"error,omitempty"`
	Untrusted  []string `json:"_untrusted,omitempty"`
}

// BatchItemError mirrors the top-level error taxonomy (code + retryable) for a
// single failed batch item, so an agent applies the same retry logic per item
// as it does for a whole-command failure.
type BatchItemError struct {
	Code      string `json:"code"`
	Retryable bool   `json:"retryable"`
	Message   string `json:"message,omitempty"`
}

// BatchItem is one entry in an aggregated batch result. Target is the input
// identifier (the natural key — the requested symbol), not an array index, so
// the agent can zip results back to inputs. On success Data carries the payload;
// on failure Error carries the per-item error and Data is nil.
type BatchItem[T any] struct {
	Target string          `json:"target"`
	OK     bool            `json:"ok"`
	Data   T               `json:"data,omitempty"`
	Error  *BatchItemError `json:"error,omitempty"`
}

// BatchSummary reports the item tally; counts always satisfy total = succeeded + failed.
type BatchSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	// Skipped counts targets not attempted because --continue-on-error=false
	// stopped the batch early; the agent can resume from these.
	Skipped int `json:"skipped,omitempty"`
}

// BatchResult is the aggregated batch envelope payload shared by every batch
// query: per-item results plus a summary. The external shape is identical for
// class A (native upstream multi-code) and class B (client-side loop).
type BatchResult[T any] struct {
	Items   []BatchItem[T] `json:"items"`
	Summary BatchSummary   `json:"summary"`
}

// classifyBatchError maps an API error onto the per-item {code, retryable}
// taxonomy used in BatchItem.Error. It mirrors the cmd-layer classifyError so a
// per-item failure carries the same code an agent would see for a whole-command
// failure of the same kind; the message is intentionally omitted by callers that
// would leak upstream URLs (already redacted at the client boundary).
func classifyBatchError(err error) *BatchItemError {
	switch {
	case errors.As(err, new(*ValidationError)):
		return &BatchItemError{Code: "E_VALIDATION", Retryable: false, Message: err.Error()}
	case errors.As(err, new(*NotFoundError)):
		return &BatchItemError{Code: "E_NOT_FOUND", Retryable: false, Message: err.Error()}
	case errors.As(err, new(*RateLimitError)):
		return &BatchItemError{Code: "E_RATE_LIMITED", Retryable: true, Message: err.Error()}
	case errors.As(err, new(*AuthError)):
		return &BatchItemError{Code: "E_AUTH", Retryable: false, Message: err.Error()}
	case errors.As(err, new(*ForbiddenError)):
		return &BatchItemError{Code: "E_FORBIDDEN", Retryable: false, Message: err.Error()}
	case errors.As(err, new(*TimeoutError)):
		return &BatchItemError{Code: "E_TIMEOUT", Retryable: true, Message: err.Error()}
	case errors.As(err, new(*ServerError)):
		return &BatchItemError{Code: "E_SERVER", Retryable: true, Message: err.Error()}
	case errors.As(err, new(*NetworkError)):
		return &BatchItemError{Code: "E_NETWORK", Retryable: true, Message: err.Error()}
	default:
		return &BatchItemError{Code: "E_UNKNOWN", Retryable: false, Message: err.Error()}
	}
}

// DepthLevel represents a single bid/ask price level.
type DepthLevel struct {
	Price *float64 `json:"price"`
	Vol   *float64 `json:"vol"`
}

// KlineBar represents a single K-line bar.
type KlineBar struct {
	Date   string   `json:"date"`
	Open   *float64 `json:"open"`
	Close  *float64 `json:"close"`
	High   *float64 `json:"high"`
	Low    *float64 `json:"low"`
	Volume *float64 `json:"volume"`
}

// MinuteTick represents a single minute-level tick.
type MinuteTick struct {
	Time   string   `json:"time"`
	Price  *float64 `json:"price"`
	Volume *float64 `json:"volume"`
	Amount *float64 `json:"amount"`
}

// SearchResult represents a stock search result.
type SearchResult struct {
	Symbol    string   `json:"symbol"`
	Name      string   `json:"name"`
	Market    string   `json:"market"`
	Pinyin    string   `json:"pinyin"`
	Untrusted []string `json:"_untrusted,omitempty"`
}

// LeadingStock is the best-performing constituent of a sector (领涨股).
type LeadingStock struct {
	Code      string   `json:"code,omitempty"`
	Name      string   `json:"name,omitempty"`
	ChangePct *float64 `json:"change_pct,omitempty"`
	Price     *float64 `json:"price,omitempty"`
	Untrusted []string `json:"_untrusted,omitempty"`
}

// Sector represents one industry/concept board ranking row.
type Sector struct {
	Code           string        `json:"code"`
	Name           string        `json:"name"`
	ChangePct      *float64      `json:"change_pct,omitempty"`
	Change         *float64      `json:"change,omitempty"`
	Price          *float64      `json:"price,omitempty"`
	Turnover       *float64      `json:"turnover,omitempty"`        // 成交额
	Volume         *float64      `json:"volume,omitempty"`          // 成交量
	TurnoverRate   *float64      `json:"turnover_rate,omitempty"`   // 换手率
	AdvanceDecline string        `json:"advance_decline,omitempty"` // 板块内涨跌家数, e.g. "190/481"
	LeadingStock   *LeadingStock `json:"leading_stock,omitempty"`
	Untrusted      []string      `json:"_untrusted,omitempty"`
}

// MarketBreadth is the advance/decline breakdown for a single exchange.
type MarketBreadth struct {
	Name      string   `json:"name"`
	Advancing int      `json:"advancing"`
	Declining int      `json:"declining"`
	Flat      int      `json:"flat"`
	Amount    *float64 `json:"amount,omitempty"`
	Untrusted []string `json:"_untrusted,omitempty"`
}

// Financials holds company fundamentals parsed from the same Tencent quote
// string the quote command uses (qt.gtimg.cn). The full ~-delimited quote line
// carries valuation fields; we surface the reliable subset. Any field absent
// from the payload stays nil. Name is external data and is listed in Untrusted.
type Financials struct {
	Symbol         string   `json:"symbol"`
	Market         string   `json:"market"`
	Name           string   `json:"name,omitempty"`
	Code           string   `json:"code,omitempty"`
	Price          *float64 `json:"price,omitempty"`
	MarketCap      *float64 `json:"market_cap,omitempty"`       // 总市值 (yuan)
	FloatMarketCap *float64 `json:"float_market_cap,omitempty"` // 流通市值 (yuan)
	PeRatio        *float64 `json:"pe_ratio,omitempty"`         // 市盈率
	Pb             *float64 `json:"pb,omitempty"`               // 市净率
	TurnoverRate   *float64 `json:"turnover_rate,omitempty"`    // 换手率 (%)
	Amount         *float64 `json:"amount,omitempty"`           // 成交额 (yuan)
	Warnings       []string `json:"warnings,omitempty"`
	Untrusted      []string `json:"_untrusted,omitempty"`
}

// MarketStats aggregates whole-market breadth statistics.
// LimitUp/LimitDown are best-effort and may be nil when the upstream pool is
// unavailable (e.g. non-trading day or pre-open); see Warnings.
type MarketStats struct {
	Advancing *int            `json:"advancing,omitempty"`
	Declining *int            `json:"declining,omitempty"`
	Flat      *int            `json:"flat,omitempty"`
	LimitUp   *int            `json:"limit_up,omitempty"`
	LimitDown *int            `json:"limit_down,omitempty"`
	Amount    *float64        `json:"amount,omitempty"` // total turnover across markets (yuan)
	Markets   []MarketBreadth `json:"markets,omitempty"`
	Warnings  []string        `json:"warnings,omitempty"`
}
