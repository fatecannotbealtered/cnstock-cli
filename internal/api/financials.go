package api

import (
	"context"
	"regexp"
	"strings"
)

// Tencent quote-string field indices for the valuation tail (0-based, after the
// regex strips the `v_<symbol>=` wrapper). Cross-referenced against the quote
// parser (which already reads amount=37, turnover=38, pe=39) and verified live
// against qt.gtimg.cn for A-shares. Market-cap fields are denominated in 亿
// (1e8 yuan), amount in 万 (1e4 yuan).
const (
	finIdxAmount       = 37 // 成交额 (万)
	finIdxTurnoverRate = 38 // 换手率 (%)
	finIdxPe           = 39 // 市盈率
	finIdxFloatCap     = 44 // 流通市值 (亿)
	finIdxMarketCap    = 45 // 总市值 (亿)
	finIdxPb           = 46 // 市净率
)

var financialsLineRe = regexp.MustCompile(`v_(\w+)="(.*)"`)

// FetchFinancials fetches company fundamentals (market cap, PE, PB, turnover,
// amount) for a symbol from the same Tencent quote string the quote command
// parses (qt.gtimg.cn). Reusing that reliable endpoint avoids the empty/EOF
// behavior of Eastmoney's stock/get from many networks.
func FetchFinancials(ctx context.Context, client *Client, symbol string) (*Financials, error) {
	normalized, err := NormalizeSymbol(symbol)
	if err != nil {
		return nil, err
	}
	text, err := client.GetString(ctx, ResolveQuoteURL(normalized))
	if err != nil {
		return nil, err
	}
	return parseFinancialsResponse(text, normalized)
}

// FetchFinancialsRaw returns the raw upstream quote string backing financials.
func FetchFinancialsRaw(ctx context.Context, client *Client, symbol string) (string, error) {
	normalized, err := NormalizeSymbol(symbol)
	if err != nil {
		return "", err
	}
	return client.GetString(ctx, ResolveQuoteURL(normalized))
}

func parseFinancialsResponse(text, symbol string) (*Financials, error) {
	match := financialsLineRe.FindStringSubmatch(text)
	// Tencent emits v_pv_none_match for an unknown symbol; treat that and any
	// missing line as not-found.
	if match == nil || match[1] == "pv_none_match" || strings.TrimSpace(match[2]) == "" {
		return nil, newNotFoundError("no fundamentals for %s", symbol)
	}

	parts := strings.Split(match[2], "~")
	market := DetectMarket(symbol)
	f := &Financials{
		Symbol:         symbol,
		Market:         MarketName[market],
		Name:           getStr(parts, 1),
		Code:           getStr(parts, 2),
		Price:          getFloat(parts, 3),
		MarketCap:      scaleFloat(getFloat(parts, finIdxMarketCap), 1e8),
		FloatMarketCap: scaleFloat(getFloat(parts, finIdxFloatCap), 1e8),
		PeRatio:        getFloat(parts, finIdxPe),
		Pb:             getFloat(parts, finIdxPb),
		TurnoverRate:   getFloat(parts, finIdxTurnoverRate),
		Amount:         scaleFloat(getFloat(parts, finIdxAmount), 1e4),
	}
	if f.Market == "" {
		f.Market = "unknown"
	}
	if f.Name != "" {
		f.Untrusted = append(f.Untrusted, "name")
	}
	return f, nil
}

// scaleFloat multiplies a float pointer by factor, preserving nil. Tencent
// reports market cap in 亿 and amount in 万; callers normalize to plain yuan.
func scaleFloat(v *float64, factor float64) *float64 {
	if v == nil {
		return nil
	}
	scaled := *v * factor
	return &scaled
}
