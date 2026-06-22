// orders.go
package arrow

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// OrderRequest represents the structure for placing an order.
type OrderRequest struct {
	Exchange         string `json:"exchange"`               // Exchange where the order is placed (e.g., NSE, BSE).
	Quantity         string `json:"quantity"`               // Order quantity.
	DisclosedQty     string `json:"disclosedQty,omitempty"` // Disclosed quantity (optional).
	Product          string `json:"product"`                // Product type (e.g., MIS, CNC, NRML).
	Symbol           string `json:"symbol"`                 // Trading symbol of the instrument.
	TransactionType  string `json:"transactionType"`        // Order transaction type (BUY/SELL).
	OrderType        string `json:"order"`                  // Type of order (e.g., MARKET, LIMIT).
	Price            string `json:"price"`                  // Order price (applicable for LIMIT orders).
	Validity         string `json:"validity"`               // Order validity (e.g., DAY, IOC).
	Remarks          string `json:"remarks,omitempty"`      // Custom Remarks for order tracking (optional).
	MarketProtection bool   `json:"mpp"`                    // Market protection flag; defaults to false when not set.
	TriggerPrice     string `json:"triggerPrice,omitempty"` // Trigger price for SL orders.
}

// OrderResponse represents the API response after placing an order.
type OrderResponse struct {
	Status    string `json:"status"`              // API response status (e.g., "success", "error").
	Message   string `json:"message,omitempty"`   // Message from the API (if any).
	ErrorCode string `json:"errorCode,omitempty"` // Error code in case of failure.
	Data      struct {
		OrderNo     string `json:"orderNo,omitempty"`     // Order number assigned by the exchange.
		RequestTime string `json:"requestTime,omitempty"` // Timestamp of the order request.
	} `json:"data,omitempty"`
}

// OrderDetails represents detailed information about an order from the order book.
//
// This struct contains comprehensive information about an order including
// execution details, pricing information, timestamps, and various order
// parameters for tracking and management purposes.
type OrderDetails struct {
	UserID             string `json:"userID"`             // Unique identifier for the user who placed the order.
	AccountID          string `json:"accountID"`          // Account ID associated with the order.
	Exchange           string `json:"exchange"`           // Exchange where the order is placed (e.g., NSE, BSE).
	Symbol             string `json:"symbol"`             // Trading symbol of the instrument.
	ID                 string `json:"id"`                 // Unique order identifier assigned by the system.
	RejectReason       string `json:"rejectReason"`       // Reason for order rejection (if applicable).
	Price              string `json:"price"`              // Order price specified by the user.
	Quantity           string `json:"quantity"`           // Total order quantity.
	MarketProtection   string `json:"marketProtection"`   // Market protection percentage applied to the order.
	Product            string `json:"product"`            // Product type (e.g., MIS, CNC, NRML).
	OrderStatus        string `json:"orderStatus"`        // Current status of the order (e.g., OPEN, COMPLETE, REJECTED).
	ReportType         string `json:"reportType"`         // Type of order report (e.g., NEW, FILL, CANCEL).
	TransactionType    string `json:"transactionType"`    // Order transaction type (BUY/SELL).
	Order              string `json:"order"`              // Type of order (e.g., MARKET, LIMIT, SL, SL-M).
	FillShares         string `json:"fillShares"`         // Number of shares filled/executed.
	AveragePrice       string `json:"averagePrice"`       // Average execution price of filled shares.
	ExchangeOrderID    string `json:"exchangeOrderID"`    // Order ID assigned by the exchange.
	CancelQuantity     string `json:"cancelQuantity"`     // Quantity that was cancelled.
	Remarks            string `json:"remarks"`            // Additional remarks or comments for the order.
	DisclosedQuantity  string `json:"disclosedQuantity"`  // Disclosed quantity for iceberg orders.
	OrderTriggerPrice  string `json:"orderTriggerPrice"`  // Trigger price for stop-loss orders.
	Validity           string `json:"validity"`           // Order validity period (e.g., DAY, IOC, GTC).
	BookProfitPrice    string `json:"bookProfitPrice"`    // Book profit price for bracket orders.
	BookLossPrice      string `json:"bookLossPrice"`      // Book loss price for bracket orders.
	TrailingPrice      string `json:"trailingPrice"`      // Trailing stop loss price.
	Amo                string `json:"amo"`                // After Market Order flag.
	PricePrecision     string `json:"pricePrecision"`     // Number of decimal places for price precision.
	TickSize           string `json:"tickSize"`           // Minimum price movement allowed for the instrument.
	LotSize            string `json:"lotSize"`            // Minimum trading quantity for the instrument.
	Token              string `json:"token"`              // Unique token identifier for the trading instrument.
	OrderTime          string `json:"orderTime"`          // Timestamp when the order was placed.
	ExchangeUpdateTime string `json:"exchangeUpdateTime"` // Last update timestamp from the exchange.
	ExchangeTime       string `json:"exchangeTime"`       // Timestamp from the exchange system.
	OrderSource        string `json:"orderSource"`        // Source from which the order was placed (e.g., WEB, MOBILE, API).
	LeavesQuantity     string `json:"leavesQuantity"`     // Remaining quantity yet to be executed.
}

type OrderDetailsResponse struct {
	Data []struct {
		Status             string `json:"status"`
		Exchange           string `json:"exchange"`
		Symbol             string `json:"symbol"`
		ID                 string `json:"id"`
		Price              string `json:"price"`
		Quantity           string `json:"quantity"`
		Product            string `json:"product"`
		OrderStatus        string `json:"orderStatus"`
		ReportType         string `json:"reportType"`
		TransactionType    string `json:"transactionType"`
		Order              string `json:"order"`
		FillShares         string `json:"fillShares"`
		AveragePrice       string `json:"averagePrice"`
		RejectReason       string `json:"rejectReason"`
		ExchangeOrderID    string `json:"exchangeOrderID"`
		CancelQuantity     string `json:"cancelQuantity"`
		Remarks            string `json:"remarks"`
		DisclosedQuantity  string `json:"disclosedQuantity"`
		OrderTriggerPrice  string `json:"orderTriggerPrice"`
		Retention          string `json:"retention"`
		BookProfitPrice    string `json:"bookProfitPrice"`
		BookLossPrice      string `json:"bookLossPrice"`
		TrailingPrice      string `json:"trailingPrice"`
		Amo                string `json:"amo"`
		PricePrecision     string `json:"pricePrecision"`
		TickSize           string `json:"tickSize"`
		LotSize            string `json:"lotSize"`
		Token              string `json:"token"`
		TimeStamp          string `json:"timeStamp"`
		OrderTime          string `json:"orderTime"`
		ExchangeUpdateTime string `json:"exchangeUpdateTime"`
		RequestTime        string `json:"requestTime"`
		ErrorMessage       string `json:"errorMessage"`
	} `json:"data"`
	Status string `json:"status"`
}

// OrderBookResponse represents the API response structure for the order book.
//
// This struct encapsulates the response from the order book API endpoint,
// containing both the order details and the API response status.
type OrderBookResponse struct {
	Data   []OrderDetails `json:"data"`   // Array of OrderDetails objects representing all user orders.
	Status string         `json:"status"` // API response status indicating success or failure.
}

// Trade is a single execution line from the trade book (/user/trades).
type Trade struct {
	Exchange        string `json:"exchange"`
	Symbol          string `json:"symbol"`
	ID              string `json:"id"`
	OrderID         string `json:"orderID"`
	TransactionType string `json:"transactionType"`
	Product         string `json:"product"`
	Quantity        string `json:"quantity"`
	FillPrice       string `json:"fillPrice"`
	FillTime        string `json:"fillTime"`
}

// PlaceOrder places a new order in the market.
//
// It sends a POST request to the API endpoint "/order/{orderType}" with the order details.
//
// Parameters:
//   - orderType: Type of order (e.g., regular for now).
//   - order: OrderRequest struct containing the order details.
//
// Returns:
//   - A pointer to OrderResponse with the order confirmation details if successful.
//   - An error if the order placement fails.
func (c *Client) PlaceOrder(orderType string, order OrderRequest) (*OrderResponse, error) {
	endpoint := fmt.Sprintf("/order/%s", orderType)

	payload, err := json.Marshal(order)
	c.debugf("Placing order", func(e *zerolog.Event) {
		e.Str("orderType", orderType)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to serialize order request")
		return nil, err
	}

	resp, err := c.request(endpoint, "POST", payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to place order")
		return nil, err
	}

	var result OrderResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		log.Error().Err(err).Msg("Failed to parse order response")
		return nil, err
	}

	if result.Status != "success" {
		log.Error().Str("errorCode", result.ErrorCode).Str("message", result.Message).Msg("Order placement failed")
		return nil, fmt.Errorf("order placement failed")
	}

	c.debugf("Order placed successfully", func(e *zerolog.Event) {
		e.Str("orderNo", result.Data.OrderNo)
	})
	return &result, nil
}

// ModifyOrder modifies an existing order.
//
// It sends a PATCH request to the API endpoint "/order/{variety}/{orderID}" with the modified order details.
//
// Parameters:
//   - variety: Order variety (e.g. "regular"), same as PlaceOrder — not LMT/MKT.
//   - orderID: Unique identifier of the order to be modified.
//   - order: OrderRequest struct containing updated order details.
//
// Returns:
//   - A pointer to OrderResponse with the updated order details if successful.
//   - An error if the modification fails.
func (c *Client) ModifyOrder(orderType, orderID string, order OrderRequest) (*OrderResponse, error) {
	endpoint := fmt.Sprintf("/order/%s/%s", orderType, orderID)

	payload, err := json.Marshal(order)
	if err != nil {
		log.Error().Err(err).Msg("Failed to serialize modify order request")
		return nil, err
	}

	resp, err := c.request(endpoint, "PATCH", payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to modify order")
		return nil, err
	}

	var result OrderResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		log.Error().Err(err).Msg("Failed to parse modify order response")
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("order modification failed")
	}

	c.debugf("Order modified successfully", func(e *zerolog.Event) {
		e.Str("orderNo", result.Data.OrderNo)
	})
	return &result, nil
}

// CancelOrder cancels an existing order.
//
// It sends a DELETE request to the API endpoint "/order/{variety}/{orderID}".
//
// Parameters:
//   - variety: Order variety (e.g. "regular"), same as PlaceOrder — not LMT/MKT.
//   - orderID: Unique identifier of the order.
//
// Returns:
//   - An error if the cancellation fails; otherwise, nil.
func (c *Client) CancelOrder(orderType, orderID string) error {
	endpoint := fmt.Sprintf("/order/%s/%s", orderType, orderID)

	resp, err := c.request(endpoint, "DELETE", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to cancel order")
		return err
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			Message string `json:"message"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		log.Error().Err(err).Msg("Failed to parse cancel order response")
		return err
	}

	if result.Status != "success" {
		return fmt.Errorf("order cancellation failed")
	}

	c.debugf("Order cancelled successfully", func(e *zerolog.Event) {
		e.Str("message", result.Data.Message)
	})
	return nil
}

// CancelAllOrders cancels all open orders (DELETE /user/orders).
func (c *Client) CancelAllOrders() error {
	resp, err := c.request("/user/orders", "DELETE", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to cancel all orders")
		return err
	}
	var result GenericResponse[map[string]any]
	if err := json.Unmarshal(resp, &result); err != nil {
		log.Error().Err(err).Msg("Failed to parse cancel all orders response")
		return err
	}
	if result.Status != "success" {
		return fmt.Errorf("cancel all orders failed with status: %s", result.Status)
	}
	c.debugf("All orders cancelled successfully", nil)
	return nil
}

// GetOrder retrieves details of a specific order.
//
// It sends a GET request to the API endpoint "/order/{orderID}".
//
// Parameters:
//   - orderID: Unique identifier of the order.
//
// Returns:
//   - A pointer to OrderDetailsResponse containing order details if successful.
//   - An error if the retrieval fails.
func (c *Client) GetOrder(orderID string) (*OrderDetailsResponse, error) {
	endpoint := fmt.Sprintf("/order/%s", orderID)

	resp, err := c.request(endpoint, "GET", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get order details")
		return nil, err
	}

	var result OrderDetailsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		log.Error().Err(err).Msg("Failed to parse order details response")
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("failed to retrieve order details")
	}

	c.debugf("Order details retrieved successfully", func(e *zerolog.Event) {
		e.Str("orderNo", orderID)
	})
	return &result, nil
}

// GetOrderBook retrieves all orders for the current trading day.
//
// This method sends a GET request to the "/user/orders" endpoint to fetch
// comprehensive order data including current orders, historical orders,
// execution details, and order status information.
//
// The returned orders include all order types (MARKET, LIMIT, SL, SL-M)
// across all exchanges and product types associated with the user's account.
// Each order contains detailed information about execution status, fill quantities,
// average prices, and various timestamps for comprehensive order tracking.
//
// Returns:
//   - A slice of OrderDetails structs containing detailed information about each order.
//   - An error if the API request fails, authentication is invalid, or response parsing fails.
//
// Example usage:
//
//	orders, err := client.GetOrderBook()
//	if err != nil {
//	    log.Printf("Failed to get order book: %v", err)
//	    return
//	}
//
//	for _, order := range orders {
//	    fmt.Printf("Order ID: %s, Symbol: %s, Status: %s, Qty: %s/%s\n",
//	               order.ID, order.Symbol, order.OrderStatus,
//	               order.FillShares, order.Quantity)
//	}
func (c *Client) GetOrderBook() ([]OrderDetails, error) {
	endpoint := "/user/orders"

	// Send a GET request to the API to fetch order book details.
	resp, err := c.request(endpoint, "GET", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch order book")
		return nil, err
	}

	var result OrderBookResponse
	// Parse the JSON response into the OrderBookResponse struct.
	if err := json.Unmarshal(resp, &result); err != nil {
		log.Error().Err(err).Msg("Failed to parse order book response")
		return nil, err
	}

	// Check if the API response status indicates success.
	if result.Status != "success" {
		return nil, fmt.Errorf("order book retrieval failed with status: %s", result.Status)
	}

	c.debugf("Order book retrieved successfully", func(e *zerolog.Event) {
		e.Int("count", len(result.Data))
	})
	return result.Data, nil
}

// GetTradeBook retrieves executed trades for the user (GET /user/trades).
func (c *Client) GetTradeBook() ([]Trade, error) {
	resp, err := c.request("/user/trades", "GET", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch trade book")
		return nil, err
	}
	var result GenericResponse[[]Trade]
	if err := json.Unmarshal(resp, &result); err != nil {
		log.Error().Err(err).Msg("Failed to parse trade book response")
		return nil, err
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("trade book retrieval failed with status: %s", result.Status)
	}
	c.debugf("Trade book retrieved successfully", func(e *zerolog.Event) {
		e.Int("count", len(result.Data))
	})
	return result.Data, nil
}
