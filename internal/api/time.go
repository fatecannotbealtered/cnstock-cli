package api

import (
	"strings"
	"time"
)

var chinaMarketLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func marketLocation(market string) *time.Location {
	switch market {
	case MarketUS:
		if loc, err := time.LoadLocation("America/New_York"); err == nil {
			return loc
		}
		return time.FixedZone("America/New_York", -5*60*60)
	default:
		return chinaMarketLocation
	}
}

func parseQuoteTimeUTC(raw, market string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) < len("20060102150405") {
		return ""
	}
	t, err := time.ParseInLocation("20060102150405", raw[:14], marketLocation(market))
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseMinuteTimeUTC(raw, market string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) != len("1504") {
		return ""
	}
	loc := marketLocation(market)
	tod, err := time.ParseInLocation("1504", raw, loc)
	if err != nil {
		return ""
	}
	now := time.Now().In(loc)
	t := time.Date(now.Year(), now.Month(), now.Day(), tod.Hour(), tod.Minute(), 0, 0, loc)
	return t.UTC().Format(time.RFC3339)
}
