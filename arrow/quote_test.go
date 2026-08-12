package arrow

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetQuotesMCXFOBatchLTP(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/info/quotes/ltp" {
			t.Fatalf("path = %s, want /info/quotes/ltp", r.URL.Path)
		}
		body, err := readBody(r)
		if err != nil {
			t.Fatal(err)
		}
		var instruments []QuoteInstrument
		if err := json.Unmarshal(body, &instruments); err != nil {
			t.Fatal(err)
		}
		if len(instruments) != 3 {
			t.Fatalf("len(instruments) = %d, want 3", len(instruments))
		}
		foundMCXFO := false
		for _, inst := range instruments {
			if inst.Exchange == "MCXFO" && inst.Symbol == "GOLDPETAL31JUL26F" {
				foundMCXFO = true
			}
		}
		if !foundMCXFO {
			t.Fatalf("expected MCXFO GOLDPETAL31JUL26F in request: %+v", instruments)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":[{"symbol":"GOLDPETAL31JUL26F","ltp":14300}]}`))
	})
	defer server.Close()

	quotes, err := client.GetQuotes([]QuoteInstrument{
		{Exchange: string(ExchangeNSE), Symbol: "ADANIENT-EQ"},
		{Exchange: string(ExchangeMCXFO), Symbol: "GOLDPETAL31JUL26F"},
		{Exchange: string(ExchangeBSE), Symbol: "RELIANCE"},
	}, InfoQuoteLTP)
	if err != nil {
		t.Fatal(err)
	}
	if len(quotes) != 1 {
		t.Fatalf("len(quotes) = %d, want 1", len(quotes))
	}
	if quotes[0]["symbol"] != "GOLDPETAL31JUL26F" {
		t.Fatalf("symbol = %v, want GOLDPETAL31JUL26F", quotes[0]["symbol"])
	}
}

func TestGetQuoteMCXFO(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/info/quote/ltp" {
			t.Fatalf("path = %s, want /info/quote/ltp", r.URL.Path)
		}
		body, err := readBody(r)
		if err != nil {
			t.Fatal(err)
		}
		var req QuoteInstrument
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatal(err)
		}
		if req.Exchange != "MCXFO" {
			t.Fatalf("exchange = %q, want MCXFO", req.Exchange)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"symbol":"GOLDPETAL31JUL26F","ltp":14300}}`))
	})
	defer server.Close()

	quote, err := client.GetQuote(ExchangeMCXFO, "GOLDPETAL31JUL26F", InfoQuoteLTP)
	if err != nil {
		t.Fatal(err)
	}
	if quote["ltp"] != float64(14300) {
		t.Fatalf("ltp = %v, want 14300", quote["ltp"])
	}
}
