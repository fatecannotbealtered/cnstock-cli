package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	defaultTimeout    = 10 * time.Second
	defaultMaxRetries = 2 // total of 3 attempts (1 initial + 2 retries)
	retryBackoff      = 250 * time.Millisecond
	tencentReferer    = "https://finance.qq.com"
	eastmoneyReferer  = "https://quote.eastmoney.com"
)

// UserAgent is the User-Agent header sent with every request.
// Transparent and identifiable — no browser impersonation.
var UserAgent = "cnstock-cli/dev (+https://github.com/fatecannotbealtered/cnstock-cli)"

// Default endpoints (Tencent Finance web endpoints — NOT official API).
const (
	QuoteEndpoint  = "https://qt.gtimg.cn/q=%s"
	KlineEndpoint  = "https://web.ifzq.gtimg.cn/appstock/app/%s/get?param=%s"
	MinuteEndpoint = "https://web.ifzq.gtimg.cn/appstock/app/minute/query?code=%s"
	SearchEndpoint = "https://smartbox.gtimg.cn/s3/?v=2&q=%s&t=all&c=1"
	// RankEndpoint returns sector/industry ranking (Tencent). Verbs: board_type, direct, count.
	RankEndpoint = "https://proxy.finance.qq.com/cgi/cgi-bin/rank/pt/getRank?board_type=%s&sort_type=priceChange&direct=%s&offset=0&count=%d"
	// BreadthEndpoint returns market advance/decline counts (Eastmoney, NOT Tencent).
	// f104=advancing, f105=declining, f106=flat, f6=turnover; summed across the three
	// composite indices to cover the whole market (Shanghai/Shenzhen/Beijing).
	BreadthEndpoint = "https://push2.eastmoney.com/webguest/api/qt/ulist.np/get?timil=1&np=1&fltt=2&invt=2&ut=fa5fd1943c7b386f172d6893dbfba10b&dect=1&intv=2&secids=1.000001,0.399106,0.899050&fields=f3,f14,f104,f105,f106,f6"
	// LimitUpEndpoint / LimitDownEndpoint return limit-up/down pools (Eastmoney). Verb: date (YYYYMMDD).
	LimitUpEndpoint   = "https://push2ex.eastmoney.com/getTopicZTPool?ut=7eea3edcaed734bea9cbfc24409ed989&dpt=wz.ztzt&Pageindex=0&pagesize=1&sort=fbt:asc&date=%s"
	LimitDownEndpoint = "https://push2ex.eastmoney.com/getTopicDTPool?ut=7eea3edcaed734bea9cbfc24409ed989&dpt=wz.ztzt&Pageindex=0&pagesize=1&sort=fund:asc&date=%s"
)

// Client is the HTTP client for Tencent Finance endpoints.
type Client struct {
	http       *http.Client
	maxRetries int
}

// NewClient creates a new API client with default settings.
func NewClient() *Client {
	return &Client{
		http:       &http.Client{Timeout: defaultTimeout},
		maxRetries: defaultMaxRetries,
	}
}

// WithTimeout returns a copy of the client with the given total request timeout.
func (c *Client) WithTimeout(d time.Duration) *Client {
	cp := *c
	cp.http = &http.Client{Timeout: d}
	return &cp
}

// resolveEndpoint returns the endpoint URL for the given key.
// Priority: env var CNS_{KEY}_ENDPOINT > default constant.
func resolveEndpoint(envKey, defaultURL string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultURL
}

// ResolveQuoteURL builds the full quote request URL.
func ResolveQuoteURL(symbols string) string {
	return fmt.Sprintf(resolveEndpoint("CNS_QUOTE_ENDPOINT", QuoteEndpoint), symbols)
}

// ResolveKlineURL builds the full kline request URL.
func ResolveKlineURL(path, param string) string {
	return fmt.Sprintf(resolveEndpoint("CNS_KLINE_ENDPOINT", KlineEndpoint), path, param)
}

// ResolveMinuteURL builds the full minute request URL.
func ResolveMinuteURL(symbol string) string {
	return fmt.Sprintf(resolveEndpoint("CNS_MINUTE_ENDPOINT", MinuteEndpoint), symbol)
}

// ResolveSearchURL builds the full search request URL.
func ResolveSearchURL(keyword string) string {
	return fmt.Sprintf(resolveEndpoint("CNS_SEARCH_ENDPOINT", SearchEndpoint), keyword)
}

// ResolveRankURL builds the full sector-ranking request URL.
func ResolveRankURL(boardType, direct string, count int) string {
	return fmt.Sprintf(resolveEndpoint("CNS_RANK_ENDPOINT", RankEndpoint), boardType, direct, count)
}

// ResolveBreadthURL builds the full market-breadth request URL.
func ResolveBreadthURL() string {
	return resolveEndpoint("CNS_BREADTH_ENDPOINT", BreadthEndpoint)
}

// ResolveLimitUpURL builds the limit-up pool request URL for the given date (YYYYMMDD).
func ResolveLimitUpURL(date string) string {
	return fmt.Sprintf(resolveEndpoint("CNS_LIMITUP_ENDPOINT", LimitUpEndpoint), date)
}

// ResolveLimitDownURL builds the limit-down pool request URL for the given date (YYYYMMDD).
func ResolveLimitDownURL(date string) string {
	return fmt.Sprintf(resolveEndpoint("CNS_LIMITDOWN_ENDPOINT", LimitDownEndpoint), date)
}

// Get performs an HTTP GET request and returns the response body as bytes.
// Transient failures (network error or HTTP 5xx) are retried up to maxRetries times.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		body, retryable, err := c.doOnce(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retryable || ctx.Err() != nil {
			return nil, err
		}
		// Exponential-ish backoff: 250ms, 500ms, ...
		select {
		case <-ctx.Done():
			return nil, newNetworkError("request canceled: %v", ctx.Err())
		case <-time.After(retryBackoff * time.Duration(1<<attempt)):
		}
	}
	return nil, lastErr
}

// doOnce performs a single HTTP attempt; the bool return reports whether the failure
// is worth retrying (network error or HTTP 5xx).
func (c *Client) doOnce(ctx context.Context, url string) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, newNetworkError("creating request for %s: %v", RedactURL(url), RedactText(err.Error()))
	}
	req.Header.Set("Referer", refererForURL(url))
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, newNetworkError("network request failed for %s: %v", RedactURL(url), RedactText(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 500 {
		return nil, true, newServerError("HTTP %d %s", resp.StatusCode, resp.Status)
	}
	// Map upstream client-error statuses onto the error taxonomy so an agent can
	// distinguish "bad symbol" (404) from "rate limited" (429) via error.code +
	// retryable, instead of every 4xx collapsing to E_NETWORK.
	switch resp.StatusCode {
	case http.StatusOK:
		// healthy; fall through to body read
	case http.StatusNotFound:
		return nil, false, newNotFoundError("HTTP %d %s", resp.StatusCode, resp.Status)
	case http.StatusTooManyRequests:
		return nil, true, newRateLimitError("HTTP %d %s", resp.StatusCode, resp.Status)
	case http.StatusUnauthorized:
		return nil, false, newAuthError("HTTP %d %s", resp.StatusCode, resp.Status)
	case http.StatusForbidden:
		return nil, false, newForbiddenError("HTTP %d %s", resp.StatusCode, resp.Status)
	default:
		return nil, false, newNetworkError("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, newNetworkError("reading response: %v", err)
	}
	return body, false, nil
}

func refererForURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return tencentReferer
	}
	switch u.Hostname() {
	case "push2.eastmoney.com", "push2ex.eastmoney.com":
		return eastmoneyReferer
	default:
		return tencentReferer
	}
}

// GetString performs an HTTP GET request and returns the response body as string.
// It tries UTF-8 first, then falls back to GB18030.
func (c *Client) GetString(ctx context.Context, url string) (string, error) {
	body, err := c.Get(ctx, url)
	if err != nil {
		return "", err
	}
	return decodeResponse(body), nil
}
