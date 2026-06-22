// constants.go
package arrow

// Exchange represents a trading exchange.
// Use these constants when specifying exchange in requests (e.g., arrow.ExchangeNSE).
type Exchange string

const (
	ExchangeNSE     Exchange = "NSE"
	ExchangeBSE     Exchange = "BSE"
	ExchangeNFO     Exchange = "NFO"
	ExchangeNCD     Exchange = "NCD"
	ExchangeBFO     Exchange = "BFO"
	ExchangeBCD     Exchange = "BCD"
	ExchangeMCX     Exchange = "MCX"
	ExchangeNSESLBM Exchange = "NSESLBM"
	ExchangeINDEX   Exchange = "INDEX"
)

// Product represents the order product type (delivery, intraday, etc.).
type Product string

const (
	ProductCNC  Product = "C" // Cash and Carry (delivery)
	ProductMIS  Product = "I" // Intraday
	ProductNRML Product = "M" // Normal (F&O)
)

// TransactionType represents buy or sell.
type TransactionType string

const (
	TransactionTypeBuy  TransactionType = "B"
	TransactionTypeSell TransactionType = "S"
)

// OrderType represents the type of order (limit, market, etc.).
type OrderType string

const (
	OrderTypeLimit   OrderType = "LMT"    // Limit order
	OrderTypeMarket  OrderType = "MKT"    // Market order
	OrderTypeSL      OrderType = "SL"     // Stop Loss (legacy)
	OrderTypeSLM     OrderType = "SL-M"   // Stop Loss Market (legacy)
	OrderTypeSLLMT   OrderType = "SL-LMT" // Stop Loss Limit (REST docs)
	OrderTypeSLMKT   OrderType = "SL-MKT" // Stop Loss Market (REST docs)
)

// Validity represents order validity period.
type Validity string

const (
	ValidityDAY Validity = "DAY" // Valid for the day
	ValidityIOC Validity = "IOC" // Immediate or Cancel
	ValidityGTC Validity = "GTC" // Good Till Cancelled
)
