package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

const maxKlineLimit = 500

// validKlinePeriods enumerates the period values accepted by the Tencent K-line endpoint.
// We validate them up-front to surface clearer errors than an empty payload from upstream.
var validKlinePeriods = map[string]struct{}{
	"day":   {},
	"week":  {},
	"month": {},
}

// FetchKline fetches historical K-line data.
func FetchKline(ctx context.Context, client *Client, symbol string, period string, limit int, adj string) ([]KlineBar, error) {
	normalized, err := NormalizeSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if limit < 1 || limit > maxKlineLimit {
		return nil, newValidationError("limit must be between 1 and %d", maxKlineLimit)
	}
	if _, ok := validKlinePeriods[period]; !ok {
		return nil, newValidationError("period only supports day, week, month")
	}

	market := DetectMarket(normalized)
	path := "fqkline"
	switch market {
	case MarketHK:
		path = "hkfqkline"
	case MarketUS:
		path = "usfqkline"
	}

	adjParam, err := NormalizeAdj(adj)
	if err != nil {
		return nil, err
	}

	param := fmt.Sprintf("%s,%s,,,%d,%s", normalized, period, limit, adjParam)
	reqURL := ResolveKlineURL(path, url.QueryEscape(param))

	text, err := client.GetString(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	return parseKlineResponse(text, normalized, market, period, adjParam)
}

func parseKlineResponse(text string, symbol, market, period, adjParam string) ([]KlineBar, error) {
	var resp struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("kline response is not valid JSON: %w", err)
	}
	if resp.Code != 0 {
		return nil, newServerError("kline API error: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(resp.Data, &dataMap); err != nil {
		return nil, fmt.Errorf("kline data is not an object")
	}

	stockRaw, ok := dataMap[symbol]
	if !ok {
		return nil, newNotFoundError("no kline data for %s", symbol)
	}

	var stockData map[string]json.RawMessage
	if err := json.Unmarshal(stockRaw, &stockData); err != nil {
		return nil, newNotFoundError("no kline data for %s", symbol)
	}

	keys := klineDataKeys(market, period, adjParam)
	var klinesRaw []json.RawMessage
	found := false
	for _, key := range keys {
		if raw, ok := stockData[key]; ok {
			if err := json.Unmarshal(raw, &klinesRaw); err == nil {
				found = true
				break
			}
		}
	}
	if !found || len(klinesRaw) == 0 {
		return nil, newNotFoundError("no kline data for %s", symbol)
	}

	var bars []KlineBar
	for _, raw := range klinesRaw {
		fields := extractKlineFields(raw)
		if len(fields) < 6 {
			continue
		}
		bars = append(bars, KlineBar{
			Date:   fields[0],
			Open:   parseOptionalFloat(fields[1]),
			Close:  parseOptionalFloat(fields[2]),
			High:   parseOptionalFloat(fields[3]),
			Low:    parseOptionalFloat(fields[4]),
			Volume: parseOptionalFloat(fields[5]),
		})
	}
	return bars, nil
}

func klineDataKeys(market, period, adjParam string) []string {
	if market == MarketUS {
		return []string{period}
	}
	if adjParam != "" {
		return []string{adjParam + period, period}
	}
	return []string{period}
}

// extractKlineFields parses a JSON array that may contain mixed types (strings,
// objects, numbers) and returns only the string elements. This handles endpoints
// like hkfqkline where bars contain extra non-string fields (e.g. {} objects).
func extractKlineFields(raw json.RawMessage) []string {
	var elements []json.RawMessage
	if err := json.Unmarshal(raw, &elements); err != nil {
		return nil
	}
	var fields []string
	for _, el := range elements {
		var s string
		if err := json.Unmarshal(el, &s); err == nil {
			fields = append(fields, s)
		}
	}
	return fields
}

// parseOptionalFloat parses s as a float64; returns nil for empty or invalid input.
// Shared by quote/kline/minute parsers.
func parseOptionalFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}
