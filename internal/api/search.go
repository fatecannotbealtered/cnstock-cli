package api

import (
	"context"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var hintRe = regexp.MustCompile(`v_hint="(.*)"`)

// FetchSearch searches for stocks by keyword (Chinese/pinyin/English).
func FetchSearch(ctx context.Context, client *Client, keyword string) ([]SearchResult, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, newValidationError("search keyword cannot be empty")
	}

	reqURL := ResolveSearchURL(url.QueryEscape(keyword))
	text, err := client.GetString(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	return parseSearchResponse(text)
}

var searchMarketMap = map[string]string{
	"sh": "A股(沪)",
	"sz": "A股(深)",
	"bj": "A股(京)",
	"hk": "港股",
	"us": "美股",
}

func parseSearchResponse(text string) ([]SearchResult, error) {
	match := hintRe.FindStringSubmatch(text)
	if match == nil || match[1] == "" {
		return nil, nil
	}

	var results []SearchResult
	for _, item := range strings.Split(match[1], "^") {
		parts := strings.Split(item, "~")
		if len(parts) < 4 {
			continue
		}
		marketCode := strings.ToLower(parts[0])
		results = append(results, SearchResult{
			Symbol: composeSearchSymbol(marketCode, parts[1]),
			Name:   decodeEscapedText(parts[2]),
			Market: searchMarketMap[marketCode],
			Pinyin: parts[3],
		})
	}
	return results, nil
}

// composeSearchSymbol assembles a Tencent-style symbol (e.g. usBRK.B, sh600519) from
// the raw market prefix and code returned by the search endpoint. Preserves the dotted
// suffix for US tickers (e.g. BRK.A / BRK.B).
func composeSearchSymbol(marketCode, rawSymbol string) string {
	if marketCode == "us" {
		return "us" + strings.ToUpper(rawSymbol)
	}
	return strings.ToLower(marketCode + rawSymbol)
}

// decodeEscapedText decodes \uXXXX escape sequences (including surrogate pairs) commonly
// seen in Tencent's search response when Chinese names are JSON-escaped.
func decodeEscapedText(s string) string {
	if !strings.Contains(s, `\u`) {
		return s
	}
	if unquoted, err := strconv.Unquote(`"` + s + `"`); err == nil {
		return unquoted
	}
	return s
}
