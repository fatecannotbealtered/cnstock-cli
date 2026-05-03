package api

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

const quoteMinField = 40

var quoteLineRe = regexp.MustCompile(`v_(\w+)="(.*)"`)

// FetchQuote fetches real-time quotes for comma-separated symbols.
func FetchQuote(ctx context.Context, client *Client, symbols string) ([]Quote, error) {
	normalized, err := NormalizeSymbols(symbols)
	if err != nil {
		return nil, err
	}

	url := ResolveQuoteURL(strings.Join(normalized, ","))
	text, err := client.GetString(ctx, url)
	if err != nil {
		return nil, err
	}

	results := parseQuoteResponse(text, normalized)
	return results, nil
}

func parseQuoteResponse(text string, normalized []string) []Quote {
	var results []Quote
	seen := make(map[string]bool)

	for _, line := range strings.Split(text, ";") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		match := quoteLineRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		symbol := match[1]
		data := match[2]
		// Match the response symbol against our requested list case-insensitively
		// (Tencent occasionally lowercases US tickers).
		canonical := matchRequested(symbol, normalized)
		if canonical == "" {
			canonical = symbol
		}
		seen[canonical] = true

		if data == "" || data == "pv_none_match" {
			results = append(results, Quote{Symbol: canonical, Error: "no data"})
			continue
		}

		parts := strings.Split(data, "~")
		market := DetectMarket(canonical)
		quote := parseQuoteFields(canonical, parts, market)
		if market != MarketUnknown && len(parts) < quoteMinField {
			quote.Warnings = []string{fmt.Sprintf("field count %d less than expected %d, schema may have changed", len(parts), quoteMinField)}
			quote.FieldCount = len(parts)
		}
		results = append(results, quote)
	}

	for _, sym := range normalized {
		if !seen[sym] {
			results = append(results, Quote{Symbol: sym, Error: "Tencent did not return data for this symbol"})
		}
	}
	return results
}

// matchRequested returns the requested form of symbol if it matches case-insensitively;
// otherwise returns "".
func matchRequested(symbol string, normalized []string) string {
	for _, n := range normalized {
		if strings.EqualFold(symbol, n) {
			return n
		}
	}
	return ""
}

func parseQuoteFields(symbol string, parts []string, market string) Quote {
	q := Quote{
		Symbol: symbol,
		Market: MarketName[market],
	}

	if market == MarketUnknown {
		q.Market = "unknown"
		q.FieldCount = len(parts)
		return q
	}

	if market == MarketUS {
		q.Name = getStr(parts, 46)
		if q.Name == "" {
			q.Name = getStr(parts, 1)
		}
	} else {
		q.Name = getStr(parts, 1)
	}
	q.Code = getStr(parts, 2)
	q.Price = getFloat(parts, 3)
	q.PrevClose = getFloat(parts, 4)
	q.Open = getFloat(parts, 5)
	q.Volume = getFloat(parts, 6)
	q.Time = getStr(parts, 30)
	q.Change = getFloat(parts, 31)
	q.ChangePct = getFloat(parts, 32)
	q.High = getFloat(parts, 33)
	q.Low = getFloat(parts, 34)
	q.Amount = getFloat(parts, 37)
	q.PeRatio = getFloat(parts, 39)

	switch market {
	case MarketCN:
		q.BuyVol = getFloat(parts, 7)
		q.SellVol = getFloat(parts, 8)
		q.Turnover = getFloat(parts, 38)
		q.Bid = parseDepth(parts, []int{9, 11, 13, 15, 17})
		q.Ask = parseDepth(parts, []int{19, 21, 23, 25, 27})
	case MarketHK:
		q.Turnover = getFloat(parts, 38)
		q.High52W = getFloat(parts, 48)
		q.Low52W = getFloat(parts, 49)
		q.NameEN = getStr(parts, 46)
		q.Currency = "HKD"
	case MarketUS:
		q.High52W = getFloat(parts, 48)
		q.Low52W = getFloat(parts, 49)
		q.Currency = getStr(parts, 35)
		if q.Currency == "" {
			q.Currency = "USD"
		}
	}

	return q
}

func parseDepth(parts []string, indexes []int) []DepthLevel {
	var levels []DepthLevel
	for _, idx := range indexes {
		price := getFloat(parts, idx)
		if price == nil {
			continue
		}
		vol := getFloat(parts, idx+1)
		levels = append(levels, DepthLevel{Price: price, Vol: vol})
	}
	return levels
}

func getStr(parts []string, index int) string {
	if index < len(parts) {
		return parts[index]
	}
	return ""
}

// getFloat returns parts[index] parsed as a float pointer, or nil when missing/invalid.
func getFloat(parts []string, index int) *float64 {
	if index >= len(parts) {
		return nil
	}
	return parseOptionalFloat(parts[index])
}
