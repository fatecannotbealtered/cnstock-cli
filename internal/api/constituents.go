package api

import (
	"context"
	"encoding/json"
	"strings"
)

// FetchConstituents lists the constituent stocks of an index or board from
// Eastmoney's clist endpoint.
//
// The index argument is the Eastmoney board code (e.g. "BK0475" for a board);
// it is passed through to the upstream `fs=b:<code>` selector. We accept it
// verbatim — resolving a friendly name like 沪深300 to a board code is a
// separate concern the caller handles.
//
// REVERSE-ENGINEERED endpoint: field IDs and the board-id format need live
// verification. Parsing is defensive — an unexpected shape degrades to
// E_NOT_FOUND/E_SERVER, and rows with absent fields keep nil values.
func FetchConstituents(ctx context.Context, client *Client, index string) ([]Constituent, error) {
	code, err := normalizeBoardCode(index)
	if err != nil {
		return nil, err
	}
	text, err := client.GetString(ctx, ResolveConstituentsURL(code))
	if err != nil {
		return nil, err
	}
	return parseConstituentsResponse(text, index)
}

// FetchConstituentsRaw returns the raw upstream constituents response.
func FetchConstituentsRaw(ctx context.Context, client *Client, index string) (string, error) {
	code, err := normalizeBoardCode(index)
	if err != nil {
		return "", err
	}
	return client.GetString(ctx, ResolveConstituentsURL(code))
}

// normalizeBoardCode validates and uppercases the board/index code.
func normalizeBoardCode(index string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(index))
	if code == "" {
		return "", newValidationError("index/board code cannot be empty")
	}
	return code, nil
}

func parseConstituentsResponse(text, index string) ([]Constituent, error) {
	var resp struct {
		Rc   int `json:"rc"`
		Data *struct {
			Total int               `json:"total"`
			Diff  []json.RawMessage `json:"diff"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, newServerError("constituents response is not valid JSON: %v", err)
	}
	if resp.Data == nil || len(resp.Data.Diff) == 0 {
		return nil, newNotFoundError("no constituents for %s", index)
	}

	out := make([]Constituent, 0, len(resp.Data.Diff))
	for _, raw := range resp.Data.Diff {
		var d map[string]json.RawMessage
		if err := json.Unmarshal(raw, &d); err != nil {
			continue // skip a malformed row rather than failing the whole list
		}
		c := Constituent{
			Code:      emField(d, "f12"),
			Name:      emField(d, "f14"),
			Price:     emFloat(d, "f2"),
			ChangePct: emFloat(d, "f3"),
			Weight:    emFloat(d, "f127"), // weight is published only for some indices
		}
		if c.Code == "" && c.Name == "" {
			continue
		}
		if c.Name != "" {
			c.Untrusted = append(c.Untrusted, "name")
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, newNotFoundError("no constituents for %s", index)
	}
	return out, nil
}
