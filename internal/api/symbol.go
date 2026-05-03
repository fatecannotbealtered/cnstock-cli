package api

import (
	"regexp"
	"strings"
)

const maxBatchSize = 50

var sixDigitRe = regexp.MustCompile(`^\d{6}$`)
var hkDigitRe = regexp.MustCompile(`^\d{1,5}$`)
var tickerRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9.\-]*$`)

// Market type constants.
const (
	MarketCN      = "cn"
	MarketHK      = "hk"
	MarketUS      = "us"
	MarketUnknown = "unknown"
)

// MarketName maps market codes to display names.
var MarketName = map[string]string{
	MarketCN: "A股",
	MarketHK: "港股",
	MarketUS: "美股",
}

// DetectMarket identifies the market from a Tencent symbol prefix.
// `bj` (Beijing Stock Exchange) is grouped under MarketCN for display purposes.
func DetectMarket(symbol string) string {
	s := strings.ToLower(symbol)
	if strings.HasPrefix(s, "hk") {
		return MarketHK
	}
	if strings.HasPrefix(s, "us") {
		return MarketUS
	}
	if strings.HasPrefix(s, "sh") || strings.HasPrefix(s, "sz") || strings.HasPrefix(s, "bj") {
		return MarketCN
	}
	return MarketUnknown
}

// NormalizeSymbol converts common inputs to Tencent quote codes.
func NormalizeSymbol(symbol string) (string, error) {
	raw := strings.TrimSpace(symbol)
	if raw == "" {
		return "", newValidationError("symbol cannot be empty")
	}

	s := strings.ReplaceAll(raw, " ", "")
	lower := strings.ToLower(s)

	if strings.HasPrefix(lower, "sh") || strings.HasPrefix(lower, "sz") || strings.HasPrefix(lower, "bj") || strings.HasPrefix(lower, "hk") {
		return lower, nil
	}
	if strings.HasPrefix(lower, "us") {
		return "us" + strings.ToUpper(s[2:]), nil
	}
	if sixDigitRe.MatchString(s) {
		return "" + cnPrefixForSixDigit(s) + s, nil
	}
	if hkDigitRe.MatchString(s) {
		return "hk" + strings.Repeat("0", 5-len(s)) + s, nil
	}
	if tickerRe.MatchString(s) {
		return "us" + strings.ToUpper(s), nil
	}
	return "", newValidationError("cannot recognize symbol: %s", symbol)
}

// cnPrefixForSixDigit picks the exchange prefix for a 6-digit A-share code.
//
// Reference: Shanghai (sh) 6/5/9 first digit; Shenzhen (sz) 0/3 first digit;
// Beijing (bj) 4/8 first digit. Anything else falls back to sz to preserve historical
// behavior; callers can still use an explicit `sh`/`sz`/`bj` prefix to override.
func cnPrefixForSixDigit(code string) string {
	switch code[0] {
	case '4', '8':
		return "bj"
	case '0', '1', '2', '3':
		return "sz"
	default:
		return "sh"
	}
}

// NormalizeSymbols normalizes comma-separated batch symbols.
func NormalizeSymbols(symbols string) ([]string, error) {
	parts := strings.Split(symbols, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		norm, err := NormalizeSymbol(p)
		if err != nil {
			return nil, err
		}
		result = append(result, norm)
	}
	if len(result) == 0 {
		return nil, newValidationError("at least one symbol is required")
	}
	if len(result) > maxBatchSize {
		return nil, newValidationError("batch query supports up to %d symbols", maxBatchSize)
	}
	return result, nil
}

// NormalizeAdj converts CLI adjustment parameter to Tencent API parameter.
func NormalizeAdj(adj string) (string, error) {
	value := strings.ToLower(adj)
	switch value {
	case "", "none", "no", "raw", "unadjusted":
		return "", nil
	case "qfq", "hfq":
		return value, nil
	default:
		return "", newValidationError("adjustment only supports qfq, hfq, none")
	}
}
