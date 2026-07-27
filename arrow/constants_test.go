package arrow

import "testing"

func TestExchangeMCXFO(t *testing.T) {
	if ExchangeMCXFO != "MCXFO" {
		t.Fatalf("ExchangeMCXFO = %q, want %q", ExchangeMCXFO, "MCXFO")
	}
	if string(ExchangeMCXFO) != "MCXFO" {
		t.Fatalf("string(ExchangeMCXFO) = %q, want %q", string(ExchangeMCXFO), "MCXFO")
	}
}

func TestHFTExchMCXFO(t *testing.T) {
	if HFTExchMCXFO != 4 {
		t.Fatalf("HFTExchMCXFO = %d, want 4", HFTExchMCXFO)
	}
}
