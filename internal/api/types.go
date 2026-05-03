package api

import "fmt"

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

// newValidationError creates a ValidationError.
func newValidationError(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

// newServerError creates a ServerError.
func newServerError(format string, args ...any) error {
	return &ServerError{Msg: fmt.Sprintf(format, args...)}
}

// newNetworkError creates a NetworkError.
func newNetworkError(format string, args ...any) error {
	return &NetworkError{Msg: fmt.Sprintf(format, args...)}
}

// newNotFoundError creates a NotFoundError.
func newNotFoundError(format string, args ...any) error {
	return &NotFoundError{Msg: fmt.Sprintf(format, args...)}
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
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Market string `json:"market"`
	Pinyin string `json:"pinyin"`
}
