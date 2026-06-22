run:
	@go run examples/example.go

test:
	TEST_QUOTE_EXCHANGE=NSE TEST_QUOTE_SYMBOL=RELIANCE-EQ go run ./examples

debug:
	SDK_DEBUG=1 make test


hft:
	HFT_MODE=full go run ./examples/hft.go
