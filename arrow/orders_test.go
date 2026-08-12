package arrow

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestPlaceOrderMCXFO(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/order/regular" {
			t.Fatalf("path = %s, want /order/regular", r.URL.Path)
		}
		body, err := readBody(r)
		if err != nil {
			t.Fatal(err)
		}
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if req["exchange"] != "MCXFO" {
			t.Fatalf("exchange = %v, want MCXFO", req["exchange"])
		}
		if req["symbol"] != "GOLDPETAL31JUL26F" {
			t.Fatalf("symbol = %v, want GOLDPETAL31JUL26F", req["symbol"])
		}
		if req["product"] != "I" {
			t.Fatalf("product = %v, want I", req["product"])
		}
		if req["order"] != "LMT" {
			t.Fatalf("order = %v, want LMT", req["order"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"orderNo":"26072701000257","requestTime":"2026-07-27T10:00:00Z"}}`))
	})
	defer server.Close()

	resp, err := client.PlaceOrder("regular", OrderRequest{
		Exchange:        string(ExchangeMCXFO),
		Quantity:        "1",
		DisclosedQty:    "0",
		Product:         string(ProductMIS),
		Symbol:          "GOLDPETAL31JUL26F",
		TransactionType: string(TransactionTypeBuy),
		OrderType:       string(OrderTypeLimit),
		Price:           "14300.0",
		Validity:        string(ValidityDAY),
		TriggerPrice:    "0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data.OrderNo != "26072701000257" {
		t.Fatalf("orderNo = %q, want 26072701000257", resp.Data.OrderNo)
	}
}

func TestModifyOrderMCXFO(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/order/regular/26072701000257" {
			t.Fatalf("path = %s, want /order/regular/26072701000257", r.URL.Path)
		}
		body, err := readBody(r)
		if err != nil {
			t.Fatal(err)
		}
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if req["exchange"] != "MCXFO" {
			t.Fatalf("exchange = %v, want MCXFO", req["exchange"])
		}
		if req["price"] != "14320.0" {
			t.Fatalf("price = %v, want 14320.0", req["price"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","message":"Order modified"}`))
	})
	defer server.Close()

	_, err := client.ModifyOrder("regular", "26072701000257", OrderRequest{
		Exchange:        string(ExchangeMCXFO),
		Quantity:        "1",
		DisclosedQty:    "0",
		Product:         string(ProductMIS),
		Symbol:          "GOLDPETAL31JUL26F",
		TransactionType: string(TransactionTypeBuy),
		OrderType:       string(OrderTypeLimit),
		Price:           "14320.0",
		Validity:        string(ValidityDAY),
		TriggerPrice:    "0",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelOrderMCXFO(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/order/regular/26072701000257" {
			t.Fatalf("path = %s, want /order/regular/26072701000257", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"message":"Order cancelled"}}`))
	})
	defer server.Close()

	if err := client.CancelOrder("regular", "26072701000257"); err != nil {
		t.Fatal(err)
	}
}
