package arrow

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetGreeksPostsExchangeAndSymbol(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/info/greeks" {
			t.Fatalf("path = %s, want /info/greeks", r.URL.Path)
		}
		body, err := readBody(r)
		if err != nil {
			t.Fatal(err)
		}
		var instruments []GreeksInstrument
		if err := json.Unmarshal(body, &instruments); err != nil {
			t.Fatal(err)
		}
		if len(instruments) != 1 {
			t.Fatalf("len(instruments) = %d, want 1", len(instruments))
		}
		if instruments[0].Exchange != "NFO" || instruments[0].Symbol != "NIFTY25AUG26C24000" {
			t.Fatalf("instrument = %+v", instruments[0])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":[{"iv":0.16,"delta":-0.8,"gamma":0.0004,"theta":-48.8,"vega":2.9}]}`))
	})
	defer server.Close()

	data, err := client.GetGreeks([]GreeksInstrument{
		{Exchange: string(ExchangeNFO), Symbol: "NIFTY25AUG26C24000"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0]["iv"] != 0.16 {
		t.Fatalf("iv = %v, want 0.16", rows[0]["iv"])
	}
}
