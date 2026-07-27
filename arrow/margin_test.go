package arrow

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetBasketMarginMCXFO(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/margin/basket" {
			t.Fatalf("path = %s, want /margin/basket", r.URL.Path)
		}
		body, err := readBody(r)
		if err != nil {
			t.Fatal(err)
		}
		var req BasketMarginRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if len(req.Orders) != 1 {
			t.Fatalf("len(orders) = %d, want 1", len(req.Orders))
		}
		order := req.Orders[0]
		if order.Exchange != ExchangeMCXFO {
			t.Fatalf("exchange = %q, want MCXFO", order.Exchange)
		}
		if order.Symbol != "GOLDPETAL31JUL26F" {
			t.Fatalf("symbol = %q, want GOLDPETAL31JUL26F", order.Symbol)
		}
		if order.Product != ProductNRML {
			t.Fatalf("product = %q, want M", order.Product)
		}
		if req.IncludePositions {
			t.Fatal("includePositions should be false")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"requiredMargin":15000.5}}`))
	})
	defer server.Close()

	data, err := client.GetBasketMargin(BasketMarginRequest{
		Orders: []MarginRequest{{
			Exchange:        ExchangeMCXFO,
			Symbol:          "GOLDPETAL31JUL26F",
			Quantity:        "1",
			Price:           "14300.0",
			Product:         ProductNRML,
			TransactionType: TransactionTypeBuy,
			Order:           OrderTypeLimit,
		}},
		IncludePositions: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if data["requiredMargin"] != 15000.5 {
		t.Fatalf("requiredMargin = %v, want 15000.5", data["requiredMargin"])
	}
}

func TestGetMarginMCXFO(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/margin/order" {
			t.Fatalf("path = %s, want /margin/order", r.URL.Path)
		}
		body, err := readBody(r)
		if err != nil {
			t.Fatal(err)
		}
		var req MarginRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if req.Exchange != ExchangeMCXFO {
			t.Fatalf("exchange = %q, want MCXFO", req.Exchange)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"requiredMargin":12000}}`))
	})
	defer server.Close()

	resp, err := client.GetMargin(MarginRequest{
		Exchange:        ExchangeMCXFO,
		Symbol:          "GOLDPETAL31JUL26F",
		Quantity:        "1",
		Price:           "14300.0",
		Product:         ProductNRML,
		TransactionType: TransactionTypeBuy,
		Order:           OrderTypeLimit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data.RequiredMargin != 12000 {
		t.Fatalf("requiredMargin = %v, want 12000", resp.Data.RequiredMargin)
	}
}
