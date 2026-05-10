package arrow

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type GenericResponse[T any] struct {
	Data   T      `json:"data"`
	Status string `json:"status"`
}

type BasketMarginRequest struct {
	Orders           []MarginRequest `json:"orders"`
	IncludePositions bool            `json:"includePositions"`
}

func (c *Client) GetBasketMargin(req BasketMarginRequest) (*MarginResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.request("/margin/basket", "POST", payload)
	if err != nil {
		return nil, err
	}
	var result MarginResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type OptionChainRequest struct {
	Underlying string   `json:"underlying"`
	Exchange   Exchange `json:"exchange"`
	Count      string   `json:"count"`
	Expiry     string   `json:"expiry"`
}

func (c *Client) GetOptionChain(req OptionChainRequest) (json.RawMessage, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.request("/info/option-chain", "POST", payload)
	if err != nil {
		return nil, err
	}
	var result GenericResponse[json.RawMessage]
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("option chain retrieval failed with status: %s", result.Status)
	}
	return result.Data, nil
}

// OptionChainSymbolsByCategory is the object under response "data" for option-chain symbol listings.
// Keys are categories (e.g. "equity", "indices"); values map listing ids such as "NSE:RELIANCE-EQ"
// or "INDEX:NIFTY" to available expiry date strings.
type OptionChainSymbolsByCategory map[string]map[string][]string

// GetAllOptionChainSymbols fetches all option-chain symbol listings and expiries (GET /info/option-chain-symbols/all).
func (c *Client) GetAllOptionChainSymbols() (OptionChainSymbolsByCategory, error) {
	resp, err := c.request("/info/option-chain-symbols/all", "GET", nil)
	if err != nil {
		return nil, err
	}
	var result GenericResponse[OptionChainSymbolsByCategory]
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("option chain symbols retrieval failed with status: %s", result.Status)
	}
	if result.Data == nil {
		return OptionChainSymbolsByCategory{}, nil
	}
	return result.Data, nil
}

type Holiday struct {
	Date     string `json:"date"`
	Exchange string `json:"exchange"`
	Name     string `json:"name"`
}

func (c *Client) GetHolidays() ([]Holiday, error) {
	resp, err := c.request("/info/holidays", "GET", nil)
	if err != nil {
		return nil, err
	}
	var result GenericResponse[[]Holiday]
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("holiday retrieval failed with status: %s", result.Status)
	}
	return result.Data, nil
}

func (c *Client) GetIndexList() ([]map[string]any, error) {
	resp, err := c.request("/info/index-list", "GET", nil)
	if err != nil {
		return nil, err
	}
	var result GenericResponse[[]map[string]any]
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("index list retrieval failed with status: %s", result.Status)
	}
	return result.Data, nil
}

type InstrumentSegment string

const (
	InstrumentSegmentAll     InstrumentSegment = "all"
	InstrumentSegmentNSE     InstrumentSegment = "nse"
	InstrumentSegmentBSE     InstrumentSegment = "bse"
	InstrumentSegmentMCX     InstrumentSegment = "mcx"
	InstrumentSegmentIndices InstrumentSegment = "indices"
)

func (c *Client) GetInstrumentsCSV(segment InstrumentSegment) (string, error) {
	path := "/" + strings.ToLower(string(segment))
	if segment == "" {
		path = "/all"
	}
	resp, err := c.request(path, "GET", nil)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (c *Client) GetInstruments(segment InstrumentSegment) ([][]string, error) {
	csvText, err := c.GetInstrumentsCSV(segment)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(strings.NewReader(csvText))
	return r.ReadAll()
}

func (c *Client) GetCandleData(exchange Exchange, token, interval, fromTimestamp, toTimestamp string, oi bool) (json.RawMessage, error) {
	base := "https://historical-api.arrow.trade"
	q := url.Values{}
	q.Set("from", fromTimestamp)
	q.Set("to", toTimestamp)
	q.Set("oi", fmt.Sprintf("%t", oi))
	endpoint := fmt.Sprintf("%s/candle/%s/%s/%s?%s", base, exchange, token, interval, q.Encode())
	resp, err := c.rawRequest(endpoint, "GET", nil)
	if err != nil {
		return nil, err
	}
	var result GenericResponse[json.RawMessage]
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("candle data retrieval failed with status: %s", result.Status)
	}
	return result.Data, nil
}
