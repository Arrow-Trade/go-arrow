# go-arrow

Official-style Go SDK for [Arrow Trade APIs](https://docs.arrow.trade/), including authentication, order management, portfolio endpoints, market data APIs, and WebSocket streams.

## Install

```bash
go get github.com/arrow-trade/go-arrow
```

## Features

- Login flows:
  - request-token auth (`Authenticate`)
  - automated credential + TOTP flow (`AutoLogin`)
- Trading APIs:
  - place/modify/cancel orders
  - order book, order details, cancel all
  - trade book
- Portfolio APIs:
  - positions, holdings, limits, user profile
- Market APIs:
  - quotes (`/info/quote`, `/info/quotes`)
  - option chain and option chain symbols
  - holidays, index list, instruments CSV
  - candle/historical data
- Streaming:
  - order updates WebSocket
  - token market data WebSocket (`ltp`, `ltpc`, `quote`, `full`)
- SDK-level debug toggle for verbose diagnostics.

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/arrow-trade/go-arrow/arrow"
)

func main() {
	client := arrow.NewClient("YOUR_APP_ID", "YOUR_APP_SECRET")

	// Optional: enable verbose SDK logs
	client.SetDebug(true)

	if err := client.AutoLogin("USER_ID", "PASSWORD", "TOTP_SECRET"); err != nil {
		log.Fatal(err)
	}

	user, err := client.GetUserDetails()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Logged in user:", user.Data.Name)
}
```

## Option Chain Symbols Example

```go
symbols, err := client.GetAllOptionChainSymbols()
if err != nil {
	log.Fatal(err)
}

// Example:
// symbols["equity"]["NSE:RELIANCE-EQ"] => []string{"26-MAY-2026", ...}
fmt.Println(symbols["indices"]["INDEX:NIFTY"])
```

## WebSocket Example

```go
streams, err := client.NewStreams()
if err != nil {
	log.Fatal(err)
}
defer streams.Close()

err = streams.DataStream.Subscribe(arrow.StreamModeLTPC, []int32{26000, 26009})
if err != nil {
	log.Fatal(err)
}
```

See runnable sample in `examples/example.go`.

## Logging / Debug

- Default SDK behavior is quiet (no verbose success/request logs).
- Use:

```go
client.SetDebug(true)
```

to enable verbose request lifecycle logs.

## Security Notes

- Never commit `.env`, secrets, API tokens, passwords, or TOTP seeds.
- Rotate credentials immediately if they are accidentally printed or leaked.

## API Reference

- Arrow API docs: [docs.arrow.trade](https://docs.arrow.trade/)
- Go package docs: [pkg.go.dev/github.com/arrow-trade/go-arrow](https://pkg.go.dev/github.com/arrow-trade/go-arrow)

## License

Add a `LICENSE` file to ensure redistributable status on pkg.go.dev.
