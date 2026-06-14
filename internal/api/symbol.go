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

// indexAlias maps friendly index aliases to their Tencent quote codes.
// Without this, bare words like "hsi" would be misread as US tickers ("usHSI").
// Keys must be lowercase; lookups are case-insensitive.
var indexAlias = map[string]string{
	// Hong Kong indices
	"hsi":      "hkHSI",    // 恒生指数 Hang Seng Index
	"hstech":   "hkHSTECH", // 恒生科技指数 Hang Seng TECH Index
	"hstec":    "hkHSTECH", // alias
	"hscei":    "hkHSCEI",  // 恒生中国企业指数 Hang Seng China Enterprises
	"hsce":     "hkHSCEI",  // alias
	"hangseng": "hkHSI",    // alias
	// A-share indices
	"sse":     "sh000001", // 上证指数 Shanghai Composite
	"szse":    "sz399001", // 深证成指 Shenzhen Component
	"chinext": "sz399006", // 创业板指 ChiNext
	"star50":  "sh000688", // 科创50 STAR 50
	"csi300":  "sh000300", // 沪深300 CSI 300
	"hs300":   "sh000300", // alias
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

	// Friendly index aliases (e.g. "hsi" -> "hkHSI") take precedence so they are not
	// misclassified as US tickers by the generic ticker rule below.
	if code, ok := indexAlias[lower]; ok {
		return code, nil
	}

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
// Reference:
//
//	Shanghai  (sh): 6xxxxx (main board), 5xxxxx (ETF/futures), 9xxxxx (B-shares)
//	Shenzhen  (sz): 0xxxxx (main board), 3xxxxx (ChiNext), 1xxxxx/2xxxxx (bonds/B-shares)
//	Beijing   (bj): 4xxxxx, 8xxxxx
//
// Callers can still use an explicit `sh`/`sz`/`bj` prefix to override.
func cnPrefixForSixDigit(code string) string {
	switch code[0] {
	case '4', '8':
		return "bj"
	case '0', '1', '2', '3':
		return "sz"
	case '5', '6', '9':
		return "sh"
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

// EastmoneySecID converts a Tencent-style symbol (e.g. "sh600519") to an
// Eastmoney secid ("1.600519"). Eastmoney prefixes Shanghai/Beijing-listed
// codes with "1." and Shenzhen with "0."; HK/US are not addressable here and
// return an error. The market prefix wins over the numeric heuristic so an
// explicit "sz"/"sh"/"bj" is always honored.
func EastmoneySecID(symbol string) (string, error) {
	normalized, err := NormalizeSymbol(symbol)
	if err != nil {
		return "", err
	}
	lower := strings.ToLower(normalized)
	switch {
	case strings.HasPrefix(lower, "sh"):
		return "1." + normalized[2:], nil
	case strings.HasPrefix(lower, "sz"):
		return "0." + normalized[2:], nil
	case strings.HasPrefix(lower, "bj"):
		return "0." + normalized[2:], nil
	default:
		return "", newValidationError("symbol %s is not an A-share code addressable via Eastmoney", symbol)
	}
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
