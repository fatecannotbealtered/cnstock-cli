package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	defaultTimeout    = 10 * time.Second
	defaultMaxRetries = 2 // total of 3 attempts (1 initial + 2 retries)
	retryBackoff      = 250 * time.Millisecond
	referer           = "https://finance.qq.com" // upstream may reject requests without Referer
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
		return nil, false, newNetworkError("creating request: %v", err)
	}
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, newNetworkError("network request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 500 {
		return nil, true, newNetworkError("HTTP %d %s", resp.StatusCode, resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, newNetworkError("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, newNetworkError("reading response: %v", err)
	}
	return body, false, nil
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
