package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// FetchMinute fetches intraday minute-level data.
func FetchMinute(ctx context.Context, client *Client, symbol string) ([]MinuteTick, error) {
	reqURL, normalized, err := minuteURL(symbol)
	if err != nil {
		return nil, err
	}
	text, err := client.GetString(ctx, reqURL)
	if err != nil {
		return nil, err
	}
	return parseMinuteResponse(text, normalized)
}

// FetchMinuteRaw returns the raw upstream minute response.
func FetchMinuteRaw(ctx context.Context, client *Client, symbol string) (string, error) {
	reqURL, _, err := minuteURL(symbol)
	if err != nil {
		return "", err
	}
	return client.GetString(ctx, reqURL)
}

// minuteURL normalizes the symbol and builds the request URL.
func minuteURL(symbol string) (string, string, error) {
	normalized, err := NormalizeSymbol(symbol)
	if err != nil {
		return "", "", err
	}
	return ResolveMinuteURL(normalized), normalized, nil
}

func parseMinuteResponse(text string, symbol string) ([]MinuteTick, error) {
	var resp struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("minute response is not valid JSON: %w", err)
	}

	stockRaw, ok := resp.Data[symbol]
	if !ok {
		return nil, newNotFoundError("no minute data for %s", symbol)
	}

	var stockData struct {
		Data struct {
			Data []string `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stockRaw, &stockData); err != nil {
		return nil, nil
	}

	market := DetectMarket(symbol)
	var ticks []MinuteTick
	for _, item := range stockData.Data.Data {
		parts := strings.Fields(item)
		if len(parts) < 3 {
			continue
		}
		tick := MinuteTick{
			Time:   parseMinuteTimeUTC(parts[0], market),
			Price:  parseOptionalFloat(parts[1]),
			Volume: parseOptionalFloat(parts[2]),
		}
		if len(parts) >= 4 {
			tick.Amount = parseOptionalFloat(parts[3])
		}
		ticks = append(ticks, tick)
	}
	return ticks, nil
}
