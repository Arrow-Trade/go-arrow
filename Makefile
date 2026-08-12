run:
	@go run examples/example.go

test:
	@go run examples/example.go

debug:
	SDK_DEBUG=1 make test


hft:
	HFT_MODE=full go run ./examples/hft.go

# Closing Auction Session full-mode check (equities + nearest ATM CE/PE).
# Runs continuously until Ctrl+C. Optional: CAS_SYMBOLS=... CAS_DURATION=120
cas:
	@go run ./examples/cas.go
