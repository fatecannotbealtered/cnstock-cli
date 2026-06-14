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

// newRateLimitError creates a RateLimitError.
func newRateLimitError(format string, args ...any) error {
	return &RateLimitError{Msg: fmt.Sprintf(format, args...)}
}

// newAuthError creates an AuthError.
func newAuthError(format string, args ...any) error {
	return &AuthError{Msg: fmt.Sprintf(format, args...)}
}

// newForbiddenError creates a ForbiddenError.
func newForbiddenError(format string, args ...any) error {
	return &ForbiddenError{Msg: fmt.Sprintf(format, args...)}
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

// Financials holds company fundamentals derived from Eastmoney's stock-get
// endpoint (push2 family). All figures are best-effort: any field absent from
// the upstream payload stays nil. Name/text fields are external data and are
// listed in Untrusted.
type Financials struct {
	Symbol         string   `json:"symbol"`
	Market         string   `json:"market"`
	Name           string   `json:"name,omitempty"`
	Code           string   `json:"code,omitempty"`
	Price          *float64 `json:"price,omitempty"`
	MarketCap      *float64 `json:"market_cap,omitempty"`       // 总市值 (yuan)
	FloatMarketCap *float64 `json:"float_market_cap,omitempty"` // 流通市值 (yuan)
	PeTTM          *float64 `json:"pe_ttm,omitempty"`           // 市盈率(动态/TTM)
	PeStatic       *float64 `json:"pe_static,omitempty"`        // 市盈率(静态)
	Pb             *float64 `json:"pb,omitempty"`               // 市净率
	Eps            *float64 `json:"eps,omitempty"`              // 每股收益
	Bvps           *float64 `json:"bvps,omitempty"`             // 每股净资产
	DividendYield  *float64 `json:"dividend_yield,omitempty"`   // 股息率 (%)
	Roe            *float64 `json:"roe,omitempty"`              // 净资产收益率 (%)
	Revenue        *float64 `json:"revenue,omitempty"`          // 营业收入 (yuan)
	NetProfit      *float64 `json:"net_profit,omitempty"`       // 净利润 (yuan)
	GrossMargin    *float64 `json:"gross_margin,omitempty"`     // 毛利率 (%)
	TotalShares    *float64 `json:"total_shares,omitempty"`     // 总股本
	FloatShares    *float64 `json:"float_shares,omitempty"`     // 流通股本
	Warnings       []string `json:"warnings,omitempty"`
	Untrusted      []string `json:"_untrusted,omitempty"`
}

// Constituent is one member of an index/board, from Eastmoney's clist
// board-constituents endpoint.
type Constituent struct {
	Code      string   `json:"code"`
	Name      string   `json:"name,omitempty"`
	Price     *float64 `json:"price,omitempty"`
	ChangePct *float64 `json:"change_pct,omitempty"` // 涨跌幅 (%)
	Weight    *float64 `json:"weight,omitempty"`     // 权重 (%), when published by upstream
	Untrusted []string `json:"_untrusted,omitempty"`
}

// MoneyFlow holds main-capital / north-bound flow figures for one symbol,
// from Eastmoney's stock fund-flow endpoint. Inflow figures are net (yuan).
type MoneyFlow struct {
	Symbol         string   `json:"symbol"`
	Market         string   `json:"market"`
	Name           string   `json:"name,omitempty"`
	Code           string   `json:"code,omitempty"`
	MainInflow     *float64 `json:"main_inflow,omitempty"`     // 主力净流入 (yuan)
	MainInflowPct  *float64 `json:"main_inflow_pct,omitempty"` // 主力净占比 (%)
	SuperInflow    *float64 `json:"super_inflow,omitempty"`    // 超大单净流入
	LargeInflow    *float64 `json:"large_inflow,omitempty"`    // 大单净流入
	MediumInflow   *float64 `json:"medium_inflow,omitempty"`   // 中单净流入
	SmallInflow    *float64 `json:"small_inflow,omitempty"`    // 小单净流入
	NorthboundFlow *float64 `json:"northbound_flow,omitempty"` // 北向净流入 (yuan), when available
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
