package api

import (
	"context"
	"encoding/json"
)

// FetchFinancials fetches company fundamentals (market cap, PE, PB, EPS, ROE,
// revenue/net-profit, etc.) for an A-share symbol from Eastmoney's stock/get
// endpoint.
//
// REVERSE-ENGINEERED endpoint: the field-ID mapping below is best-effort and
// needs live verification. Parsing is defensive — an unexpected shape degrades
// to E_NOT_FOUND/E_SERVER rather than panicking, and absent fields stay nil.
func FetchFinancials(ctx context.Context, client *Client, symbol string) (*Financials, error) {
	secid, err := EastmoneySecID(symbol)
	if err != nil {
		return nil, err
	}
	text, err := client.GetString(ctx, ResolveFinancialsURL(secid))
	if err != nil {
		return nil, err
	}
	return parseFinancialsResponse(text, symbol)
}

// FetchFinancialsRaw returns the raw upstream fundamentals response.
func FetchFinancialsRaw(ctx context.Context, client *Client, symbol string) (string, error) {
	secid, err := EastmoneySecID(symbol)
	if err != nil {
		return "", err
	}
	return client.GetString(ctx, ResolveFinancialsURL(secid))
}

func parseFinancialsResponse(text, symbol string) (*Financials, error) {
	// Eastmoney stock/get wraps everything in a "data" object; numeric fields are
	// f-coded. Strings (name/code) ride as JSON strings, numbers as JSON numbers.
	var resp struct {
		Rc   int             `json:"rc"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, newServerError("financials response is not valid JSON: %v", err)
	}
	// Upstream returns data:null for an unknown secid.
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, newNotFoundError("no fundamentals for %s", symbol)
	}

	var d map[string]json.RawMessage
	if err := json.Unmarshal(resp.Data, &d); err != nil {
		return nil, newServerError("financials data is not an object")
	}

	market := DetectMarket(func() string { s, _ := NormalizeSymbol(symbol); return s }())
	f := &Financials{
		Symbol:         symbol,
		Market:         MarketName[market],
		Name:           emField(d, "f58"),
		Code:           emField(d, "f57"),
		Price:          emFloat(d, "f43"),
		MarketCap:      emFloat(d, "f116"),
		FloatMarketCap: emFloat(d, "f117"),
		PeTTM:          emFloat(d, "f162"),
		PeStatic:       emFloat(d, "f163"),
		Pb:             emFloat(d, "f167"),
		Eps:            emFloat(d, "f55"),
		Bvps:           emFloat(d, "f92"),
		DividendYield:  emFloat(d, "f173"),
		Roe:            emFloat(d, "f105"),
		Revenue:        emFloat(d, "f183"),
		NetProfit:      emFloat(d, "f186"),
		GrossMargin:    emFloat(d, "f164"),
		TotalShares:    emFloat(d, "f84"),
		FloatShares:    emFloat(d, "f85"),
	}
	if f.Market == "" {
		f.Market = "unknown"
	}
	if f.Name != "" {
		f.Untrusted = append(f.Untrusted, "name")
	}
	return f, nil
}

// emField returns a string-or-number f-coded field as a string ("" when absent
// or when upstream uses its "-" sentinel for no-data).
func emField(d map[string]json.RawMessage, key string) string {
	raw, ok := d[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "-" {
			return ""
		}
		return s
	}
	// Fall back to a bare number rendered as text.
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.String()
	}
	return ""
}

// emFloat returns an f-coded numeric field as a float pointer. Eastmoney encodes
// "no data" as the JSON string "-" or numeric sentinel; both yield nil.
func emFloat(d map[string]json.RawMessage, key string) *float64 {
	raw, ok := d[key]
	if !ok {
		return nil
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return &f
	}
	// Numbers occasionally arrive as quoted strings.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return parseOptionalFloat(s)
	}
	return nil
}
