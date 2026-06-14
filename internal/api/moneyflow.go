package api

import (
	"context"
	"encoding/json"
)

// FetchMoneyFlow fetches main-capital / north-bound money-flow figures for an
// A-share symbol from Eastmoney's stock/get fund-flow fields.
//
// REVERSE-ENGINEERED endpoint: the field-ID mapping below is best-effort and
// needs live verification. Parsing is defensive — an unexpected shape degrades
// to E_NOT_FOUND/E_SERVER, and absent figures stay nil.
func FetchMoneyFlow(ctx context.Context, client *Client, symbol string) (*MoneyFlow, error) {
	secid, err := EastmoneySecID(symbol)
	if err != nil {
		return nil, err
	}
	text, err := client.GetString(ctx, ResolveMoneyFlowURL(secid))
	if err != nil {
		return nil, err
	}
	return parseMoneyFlowResponse(text, symbol)
}

// FetchMoneyFlowRaw returns the raw upstream money-flow response.
func FetchMoneyFlowRaw(ctx context.Context, client *Client, symbol string) (string, error) {
	secid, err := EastmoneySecID(symbol)
	if err != nil {
		return "", err
	}
	return client.GetString(ctx, ResolveMoneyFlowURL(secid))
}

func parseMoneyFlowResponse(text, symbol string) (*MoneyFlow, error) {
	var resp struct {
		Rc   int             `json:"rc"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, newServerError("money-flow response is not valid JSON: %v", err)
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, newNotFoundError("no money-flow data for %s", symbol)
	}

	var d map[string]json.RawMessage
	if err := json.Unmarshal(resp.Data, &d); err != nil {
		return nil, newServerError("money-flow data is not an object")
	}

	market := DetectMarket(func() string { s, _ := NormalizeSymbol(symbol); return s }())
	mf := &MoneyFlow{
		Symbol:         symbol,
		Market:         MarketName[market],
		Name:           emField(d, "f58"),
		Code:           emField(d, "f57"),
		MainInflow:     emFloat(d, "f135"),
		MainInflowPct:  emFloat(d, "f136"),
		SuperInflow:    emFloat(d, "f137"),
		LargeInflow:    emFloat(d, "f139"),
		MediumInflow:   emFloat(d, "f141"),
		SmallInflow:    emFloat(d, "f143"),
		NorthboundFlow: emFloat(d, "f148"),
	}
	if mf.Market == "" {
		mf.Market = "unknown"
	}
	if mf.Name != "" {
		mf.Untrusted = append(mf.Untrusted, "name")
	}
	return mf, nil
}
