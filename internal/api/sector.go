package api

import (
	"context"
	"encoding/json"
	"fmt"
)

const maxSectorCount = 50

// validBoardTypes enumerates the board categories accepted by the ranking endpoint.
var validBoardTypes = map[string]struct{}{
	"hy": {}, // 行业 industry
	"gn": {}, // 概念 concept
	"dy": {}, // 地域 region
}

// FetchSectors fetches the top-N industry/concept boards ranked by change percent.
//
// direction is the user-facing intent: "up" = top gainers, "down" = top losers.
// NOTE: the upstream `direct` parameter is inverted relative to intuition —
// upstream direct=down returns gainers (descending), direct=up returns losers
// (ascending) — so we translate here to keep the CLI semantics natural.
func FetchSectors(ctx context.Context, client *Client, boardType, direction string, count int) ([]Sector, error) {
	reqURL, err := sectorURL(boardType, direction, count)
	if err != nil {
		return nil, err
	}
	text, err := client.GetString(ctx, reqURL)
	if err != nil {
		return nil, err
	}
	return parseSectorResponse(text)
}

// FetchSectorsRaw returns the raw upstream sector-ranking response.
func FetchSectorsRaw(ctx context.Context, client *Client, boardType, direction string, count int) (string, error) {
	reqURL, err := sectorURL(boardType, direction, count)
	if err != nil {
		return "", err
	}
	return client.GetString(ctx, reqURL)
}

// sectorURL validates inputs and builds the ranking request URL.
//
// direction is the user-facing intent: "up" = top gainers, "down" = top losers.
// NOTE: the upstream `direct` parameter is inverted relative to intuition —
// upstream direct=down returns gainers (descending), direct=up returns losers
// (ascending) — so we translate here to keep the CLI semantics natural.
func sectorURL(boardType, direction string, count int) (string, error) {
	if _, ok := validBoardTypes[boardType]; !ok {
		return "", newValidationError("board only supports hy (industry), gn (concept), dy (region)")
	}
	if count < 1 || count > maxSectorCount {
		return "", newValidationError("top must be between 1 and %d", maxSectorCount)
	}

	var direct string
	switch direction {
	case "up", "gainers", "":
		direct = "down" // upstream: descending by change -> gainers first
	case "down", "losers":
		direct = "up" // upstream: ascending by change -> losers first
	default:
		return "", newValidationError("direction only supports up (gainers) or down (losers)")
	}
	return ResolveRankURL(boardType, direct, count), nil
}

func parseSectorResponse(text string) ([]Sector, error) {
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			RankList []struct {
				Code     string `json:"code"`
				Name     string `json:"name"`
				Zdf      string `json:"zdf"`
				Zd       string `json:"zd"`
				Zxj      string `json:"zxj"`
				Turnover string `json:"turnover"`
				Volume   string `json:"volume"`
				Hsl      string `json:"hsl"`
				Zgb      string `json:"zgb"`
				Lzg      struct {
					Code string `json:"code"`
					Name string `json:"name"`
					Zdf  string `json:"zdf"`
					Zxj  string `json:"zxj"`
				} `json:"lzg"`
			} `json:"rank_list"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("sector response is not valid JSON: %w", err)
	}
	if resp.Code != 0 {
		return nil, newServerError("sector API error: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	sectors := make([]Sector, 0, len(resp.Data.RankList))
	for _, r := range resp.Data.RankList {
		s := Sector{
			Code:           r.Code,
			Name:           r.Name,
			ChangePct:      parseOptionalFloat(r.Zdf),
			Change:         parseOptionalFloat(r.Zd),
			Price:          parseOptionalFloat(r.Zxj),
			Turnover:       parseOptionalFloat(r.Turnover),
			Volume:         parseOptionalFloat(r.Volume),
			TurnoverRate:   parseOptionalFloat(r.Hsl),
			AdvanceDecline: r.Zgb,
			Untrusted:      []string{"name", "advance_decline"},
		}
		if r.Lzg.Name != "" {
			s.LeadingStock = &LeadingStock{
				Code:      r.Lzg.Code,
				Name:      r.Lzg.Name,
				ChangePct: parseOptionalFloat(r.Lzg.Zdf),
				Price:     parseOptionalFloat(r.Lzg.Zxj),
				Untrusted: []string{"name"},
			}
			s.Untrusted = append(s.Untrusted, "leading_stock.name")
		}
		sectors = append(sectors, s)
	}
	return sectors, nil
}
