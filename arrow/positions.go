// positions.go
package arrow

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Position represents a trading position held by the user.
//
// This struct contains comprehensive information about a trading position,
// including quantity details, pricing information, profit/loss calculations,
// and various trading metrics for both current day and carry-forward transactions.
type Position struct {
	UserID                   string `json:"userID"`                   // Unique identifier for the user holding the position.
	AccountID                string `json:"accountID"`                // Account ID associated with the position.
	Token                    string `json:"token"`                    // Unique token identifier for the trading instrument.
	Exchange                 string `json:"exchange"`                 // Name of the exchange where the instrument is traded (e.g., NSE, BSE).
	Symbol                   string `json:"symbol"`                   // Trading symbol of the instrument (e.g., RELIANCE, TCS).
	Segment                  string `json:"segment"`                  // Segment code (e.g., CM, FO, MCX).
	Product                  string `json:"product"`                  // Product type (e.g., MIS, CNC, NRML).
	Qty                      string `json:"qty"`                      // Net quantity of the position (positive for long, negative for short).
	AvgPrice                 string `json:"avgPrice"`                 // Average price at which the position was acquired.
	DayBuyQty                string `json:"dayBuyQty"`                // Quantity bought during the current trading day.
	DaySellQty               string `json:"daySellQty"`               // Quantity sold during the current trading day.
	DayBuyAmount             string `json:"dayBuyAmount"`             // Total amount spent on buying during the current day.
	DayBuyAvgPrice           string `json:"dayBuyAvgPrice"`           // Average price of buy transactions for the current day.
	DaySellAmount            string `json:"daySellAmount"`            // Total amount received from selling during the current day.
	DaySellAvgPrice          string `json:"daySellAvgPrice"`          // Average price of sell transactions for the current day.
	CarryForwardBuyQty       string `json:"carryForwardBuyQty"`       // Quantity bought and carried forward from previous sessions.
	CarryForwardSellQty      string `json:"carryForwardSellQty"`      // Quantity sold and carried forward from previous sessions.
	CarryForwardBuyAmount    string `json:"carryForwardBuyAmount"`    // Total amount of carried forward buy transactions.
	CarryForwardBuyAvgPrice  string `json:"carryForwardBuyAvgPrice"`  // Average price of carried forward buy transactions.
	CarryForwardSellAmount   string `json:"carryForwardSellAmount"`   // Total amount of carried forward sell transactions.
	CarryForwardSellAvgPrice string `json:"carryForwardSellAvgPrice"` // Average price of carried forward sell transactions.
	CarryForwardAvgPrice     string `json:"carryForwardAvgPrice"`     // Average price of all carried forward transactions.
	Ltp                      string `json:"ltp"`                      // Last traded price of the instrument (paise).
	Close                    string `json:"close"`                    // Previous close (paise).
	OptionType               string `json:"optionType"`               // Option type when applicable (CE/PE).
	RealisedPnL              string `json:"realisedPnL"`              // Realized profit and loss from closed positions.
	UnrealisedMarkToMarket   string `json:"unrealisedMarkToMarket"`   // Unrealized profit and loss based on current market price.
	BreakEvenPrice           string `json:"breakEvenPrice"`           // Price at which the position would break even.
	OpenBuyQty               string `json:"openBuyQty"`               // Outstanding buy quantity yet to be settled.
	OpenSellQty              string `json:"openSellQty"`              // Outstanding sell quantity yet to be settled.
	OpenBuyAmount            string `json:"openBuyAmount"`            // Total amount of outstanding buy transactions.
	OpenSellAmount           string `json:"openSellAmount"`           // Total amount of outstanding sell transactions.
	OpenBuyAvgPrice          string `json:"openBuyAvgPrice"`          // Average price of outstanding buy transactions.
	OpenSellAvgPrice         string `json:"openSellAvgPrice"`         // Average price of outstanding sell transactions.
	Multiplier               string `json:"multiplier"`               // Contract multiplier for derivative instruments.
	PricePrecision           string `json:"pricePrecision"`           // Number of decimal places for price precision.
	PriceFactor              string `json:"priceFactor"`              // Factor used for price calculations.
	TickSize                 string `json:"tickSize"`                 // Minimum price movement allowed for the instrument.
	LotSize                  string `json:"lotSize"`                  // Minimum trading quantity for the instrument.
	UploadPrice              string `json:"uploadPrice"`              // Price used for position upload calculations.
	NetUploadPrice           string `json:"netUploadPrice"`           // Net price after adjustments for upload calculations.
	RequestTime              string `json:"requestTime"`              // Timestamp when the position data was requested.
}

// PositionsResponse represents the API response structure for user positions.
//
// This struct encapsulates the response from the positions API endpoint,
// containing both the position data and the API response status.
type PositionsResponse struct {
	Data   []Position `json:"data"`   // Array of Position objects representing all user positions.
	Status string     `json:"status"` // API response status indicating success or failure.
}

// GetPositions retrieves all trading positions for the authenticated user.
//
// This method sends a GET request to the "/user/positions" endpoint to fetch
// comprehensive position data including current holdings, day trading activities,
// carry-forward positions, and profit/loss calculations.
//
// The returned positions include both equity and derivative instruments across
// all exchanges and product types associated with the user's account.
//
// Returns:
//   - A slice of Position structs containing detailed information about each position.
//   - An error if the API request fails, authentication is invalid, or response parsing fails.
//
// Example usage:
//
//	positions, err := client.GetPositions()
//	if err != nil {
//	    log.Printf("Failed to get positions: %v", err)
//	    return
//	}
//
//	for _, position := range positions {
//	    fmt.Printf("Symbol: %s, Qty: %s, PnL: %s\n",
//	               position.Symbol, position.Qty, position.UnrealisedMarkToMarket)
//	}
func (c *Client) GetPositions() ([]Position, error) {
	endpoint := "/user/positions"

	// Send a GET request to the API to fetch position details.
	resp, err := c.request(endpoint, "GET", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch positions")
		return nil, err
	}

	var result PositionsResponse
	// Parse the JSON response into the PositionsResponse struct.
	if err := json.Unmarshal(resp, &result); err != nil {
		log.Error().Err(err).Msg("Failed to parse positions response")
		return nil, err
	}

	// Check if the API response status indicates success.
	if result.Status != "success" {
		return nil, fmt.Errorf("positions retrieval failed with status: %s", result.Status)
	}

	c.debugf("Positions retrieved successfully", func(e *zerolog.Event) {
		e.Int("count", len(result.Data))
	})
	return result.Data, nil
}
