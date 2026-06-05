package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// FetchMarketStats aggregates whole-market breadth statistics.
//
// Advance/decline/flat counts and total turnover come from Eastmoney's ulist.np
// endpoint (summed across the Shanghai/Shenzhen/Beijing composite indices).
// Limit-up/down counts are best-effort: a failure there degrades to a warning
// rather than failing the whole command.
func FetchMarketStats(ctx context.Context, client *Client) (*MarketStats, error) {
	text, err := client.GetString(ctx, ResolveBreadthURL())
	if err != nil {
		return nil, err
	}
	stats, err := parseBreadthResponse(text)
	if err != nil {
		return nil, err
	}

	date := time.Now().Format("20060102")
	if up, err := fetchPoolCount(ctx, client, ResolveLimitUpURL(date)); err == nil {
		stats.LimitUp = up
	} else {
		stats.Warnings = append(stats.Warnings, "limit-up count unavailable: "+err.Error())
	}
	if down, err := fetchPoolCount(ctx, client, ResolveLimitDownURL(date)); err == nil {
		stats.LimitDown = down
	} else {
		stats.Warnings = append(stats.Warnings, "limit-down count unavailable: "+err.Error())
	}

	return stats, nil
}

// FetchMarketStatsRaw returns the raw upstream market-breadth response
// (the advance/decline payload; limit-up/down pools are omitted in raw mode).
func FetchMarketStatsRaw(ctx context.Context, client *Client) (string, error) {
	return client.GetString(ctx, ResolveBreadthURL())
}

func parseBreadthResponse(text string) (*MarketStats, error) {
	var resp struct {
		Rc   int `json:"rc"`
		Data struct {
			Diff []struct {
				F3   *float64 `json:"f3"`
				F6   *float64 `json:"f6"`
				F14  string   `json:"f14"`
				F104 *int     `json:"f104"`
				F105 *int     `json:"f105"`
				F106 *int     `json:"f106"`
			} `json:"diff"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("market breadth response is not valid JSON: %w", err)
	}
	if resp.Rc != 0 || len(resp.Data.Diff) == 0 {
		return nil, newServerError("market breadth API returned no data (rc=%d)", resp.Rc)
	}

	stats := &MarketStats{}
	var adv, dec, flat int
	var amount float64
	var haveAmount bool
	for _, d := range resp.Data.Diff {
		mb := MarketBreadth{Name: d.F14, Amount: d.F6}
		if d.F104 != nil {
			mb.Advancing = *d.F104
			adv += *d.F104
		}
		if d.F105 != nil {
			mb.Declining = *d.F105
			dec += *d.F105
		}
		if d.F106 != nil {
			mb.Flat = *d.F106
			flat += *d.F106
		}
		if d.F6 != nil {
			amount += *d.F6
			haveAmount = true
		}
		stats.Markets = append(stats.Markets, mb)
	}
	stats.Advancing = &adv
	stats.Declining = &dec
	stats.Flat = &flat
	if haveAmount {
		stats.Amount = &amount
	}
	return stats, nil
}

// fetchPoolCount reads the `data.tc` count from an Eastmoney ZT/DT pool endpoint.
func fetchPoolCount(ctx context.Context, client *Client, url string) (*int, error) {
	text, err := client.GetString(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Rc   int `json:"rc"`
		Data *struct {
			Tc int `json:"tc"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("pool response is not valid JSON: %w", err)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("pool returned no data (rc=%d)", resp.Rc)
	}
	tc := resp.Data.Tc
	return &tc, nil
}
