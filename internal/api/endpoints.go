package api

import (
	"net/url"
	"os"
	"regexp"
	"strings"
)

// EndpointInfo describes one configurable endpoint and whether it has been
// overridden via its environment variable.
type EndpointInfo struct {
	Name       string `json:"name"`
	Env        string `json:"env"`
	Default    string `json:"default"`
	Active     string `json:"active"`
	Overridden bool   `json:"overridden"`
}

// ProbeTarget is a ready-to-GET URL used by connectivity checks.
type ProbeTarget struct {
	Name string
	URL  string
}

type endpointDef struct {
	name string
	env  string
	def  string
}

var sensitiveURLParam = regexp.MustCompile(`(?i)^(ut)$|access[_-]?token|auth|authorization|cookie|key|passwd|password|secret|session|sign|token`)
var urlUserInfo = regexp.MustCompile(`(https?://)[^/\s"@]+@`)

// endpointDefs is the single source of truth for configurable endpoints.
var endpointDefs = []endpointDef{
	{"quote", "CNS_QUOTE_ENDPOINT", QuoteEndpoint},
	{"kline", "CNS_KLINE_ENDPOINT", KlineEndpoint},
	{"minute", "CNS_MINUTE_ENDPOINT", MinuteEndpoint},
	{"search", "CNS_SEARCH_ENDPOINT", SearchEndpoint},
	{"rank", "CNS_RANK_ENDPOINT", RankEndpoint},
	{"breadth", "CNS_BREADTH_ENDPOINT", BreadthEndpoint},
	{"limit_up", "CNS_LIMITUP_ENDPOINT", LimitUpEndpoint},
	{"limit_down", "CNS_LIMITDOWN_ENDPOINT", LimitDownEndpoint},
	{"financials", "CNS_FINANCIALS_ENDPOINT", FinancialsEndpoint},
	{"constituents", "CNS_CONSTITUENTS_ENDPOINT", ConstituentsEndpoint},
	{"moneyflow", "CNS_MONEYFLOW_ENDPOINT", MoneyFlowEndpoint},
}

// Endpoints returns metadata for every configurable endpoint, including whether
// each has been overridden by its environment variable.
func Endpoints() []EndpointInfo {
	infos := make([]EndpointInfo, 0, len(endpointDefs))
	for _, d := range endpointDefs {
		active := d.def
		overridden := false
		if v := os.Getenv(d.env); v != "" {
			active = v
			overridden = true
		}
		infos = append(infos, EndpointInfo{
			Name:       d.name,
			Env:        d.env,
			Default:    RedactURL(d.def),
			Active:     RedactURL(active),
			Overridden: overridden,
		})
	}
	return infos
}

// RedactURL removes likely credentials from endpoint URLs before they are
// exposed through context, doctor, or error details.
func RedactURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return redactSensitivePairs(raw)
	}
	if u.User != nil {
		u.User = url.User("REDACTED")
	}
	q := u.Query()
	for key := range q {
		if sensitiveURLParam.MatchString(key) {
			q.Set(key, "REDACTED")
		}
	}
	u.RawQuery = q.Encode()
	return redactSensitivePairs(u.String())
}

// RedactText removes likely credentials from diagnostic text before it is
// placed in error messages.
func RedactText(s string) string {
	s = urlUserInfo.ReplaceAllString(s, "${1}REDACTED@")
	return redactSensitivePairs(s)
}

func redactSensitivePairs(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '&' || r == '?' || r == ';'
	})
	for _, part := range parts {
		if i := strings.Index(part, "="); i > 0 && sensitiveURLParam.MatchString(part[:i]) {
			s = strings.ReplaceAll(s, part, part[:i+1]+"REDACTED")
		}
	}
	return s
}

// ProbeTargets returns representative, requestable URLs for connectivity checks.
func ProbeTargets() []ProbeTarget {
	return []ProbeTarget{
		{"quote", ResolveQuoteURL("sh000001")},
		{"kline", ResolveKlineURL("fqkline", url.QueryEscape("sh000001,day,,,1,"))},
		{"minute", ResolveMinuteURL("sh000001")},
		{"search", ResolveSearchURL(url.QueryEscape("test"))},
		{"rank", ResolveRankURL("hy", "down", 1)},
		{"breadth", ResolveBreadthURL()},
		{"financials", ResolveFinancialsURL("1.600519")},
		{"moneyflow", ResolveMoneyFlowURL("1.600519")},
	}
}
