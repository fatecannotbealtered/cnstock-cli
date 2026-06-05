package api

import (
	"net/url"
	"os"
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
			Default:    d.def,
			Active:     active,
			Overridden: overridden,
		})
	}
	return infos
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
	}
}
