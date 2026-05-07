package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Arrow-Trade/go-arrow/arrow"
	"github.com/joho/godotenv"
)

// Env / flags for local testing:
//   USER_ID, PASSWORD, TOTP_KEY, APP_ID, APP_SECRET — required for AutoLogin
//   SDK_DEBUG=1 — enable verbose SDK logs
//   SKIP_STREAMS=1 — only run REST calls (no WebSockets)
//   STREAM_DURATION — how long to listen on streams (default 15s), e.g. 30s, 1m
//   STREAM_TOKENS — comma-separated instrument tokens for market data, default 26000,26009 (Nifty 50, Bank Nifty index tokens)
//   TEST_QUOTE_EXCHANGE + TEST_QUOTE_SYMBOL — if both set, calls REST GetQuote (InfoQuoteLTP) after login

func main() {
	godotenv.Load()

	noStreams := flag.Bool("no-streams", false, "skip order + market WebSocket test")
	streamSec := flag.Int("stream-sec", 0, "seconds to run streams (0 = use STREAM_DURATION env or 15)")
	flag.Parse()

	userID := os.Getenv("USER_ID")
	password := os.Getenv("PASSWORD")
	totpKey := os.Getenv("TOTP_KEY")
	appID := os.Getenv("APP_ID")
	appSecret := os.Getenv("APP_SECRET")

	if userID == "" || password == "" || totpKey == "" || appID == "" || appSecret == "" {
		fmt.Println("Set USER_ID, PASSWORD, TOTP_KEY, APP_ID, APP_SECRET (e.g. in .env)")
		os.Exit(1)
	}
	fmt.Printf("Logging in as user=%s appID=%s\n", userID, appID)

	client := arrow.NewClient(appID, appSecret)
	if os.Getenv("SDK_DEBUG") == "1" || strings.EqualFold(os.Getenv("SDK_DEBUG"), "true") {
		client.SetDebug(true)
	}

	err := client.AutoLogin(userID, password, totpKey)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Login successful!")
	fmt.Println(client.GetToken())

	// Get user details
	user, err := client.GetUserDetails()

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("User Details: %+v\n", user)

	orders, err := client.GetOrderBook()

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Order Book: %+v\n", orders)

	holdings, err := client.GetHoldings()

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Holdings: %+v\n", holdings)

	limits, err := client.GetLimits()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Limits: %+v\n", limits)

	marginRequest := arrow.MarginRequest{
		Exchange:         arrow.ExchangeNSE,
		Symbol:           "YESBANK-EQ",
		Quantity:         "1",
		Price:            "2500",
		Product:          arrow.ProductCNC,
		TransactionType:  arrow.TransactionTypeBuy,
		Order:            arrow.OrderTypeLimit,
		IncludePositions: false,
	}

	margin, err := client.GetMargin(marginRequest)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Margin: %+v\n", margin)

	trades, err := client.GetTradeBook()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Trades: %+v\n", trades)

	qEx := os.Getenv("TEST_QUOTE_EXCHANGE")
	qSym := os.Getenv("TEST_QUOTE_SYMBOL")
	if qEx != "" && qSym != "" {
		quote, qerr := client.GetQuote(arrow.Exchange(qEx), qSym, arrow.InfoQuoteLTP)
		if qerr != nil {
			fmt.Println("GetQuote error:", qerr)
		} else {
			fmt.Printf("REST quote (%s %s): %+v\n", qEx, qSym, quote)
		}
	}

	skipStreams := *noStreams || os.Getenv("SKIP_STREAMS") == "1" || strings.EqualFold(os.Getenv("SKIP_STREAMS"), "true")
	if skipStreams {
		fmt.Println("Skipping WebSockets (use default or unset SKIP_STREAMS / omit -no-streams to test streams).")
		return
	}

	dur := 15 * time.Second
	if *streamSec > 0 {
		dur = time.Duration(*streamSec) * time.Second
	} else if d := strings.TrimSpace(os.Getenv("STREAM_DURATION")); d != "" {
		if parsed, perr := time.ParseDuration(d); perr == nil {
			dur = parsed
		}
	}

	tokens := parseStreamTokens(os.Getenv("STREAM_TOKENS"), []int32{26000, 26009})
	fmt.Printf("Connecting streams for %s (tokens %v)...\n", dur, tokens)

	streams, err := client.NewStreams()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer streams.Close()

	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	go streams.OrderStream.ReadUpdates(ctx, func(update map[string]any) {
		fmt.Printf("Order update: %+v\n", update)
	}, func(err error) {
		fmt.Println("order stream error:", err)
	})

	err = streams.DataStream.Subscribe(arrow.StreamModeLTPC, tokens)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	streams.DataStream.ReadTicks(ctx, func(tick arrow.MarketTick) {
		fmt.Printf("Tick: token=%d ltp=%d change=%.2f%% mode=%s\n", tick.Token, tick.LTP, tick.NetChange, tick.Mode)
	}, func(err error) {
		fmt.Println("data stream error:", err)
	})

	fmt.Println("Stream window finished.")
}

func parseStreamTokens(raw string, defaults []int32) []int32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		out := make([]int32, len(defaults))
		copy(out, defaults)
		return out
	}
	parts := strings.Split(raw, ",")
	var out []int32
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseInt(p, 10, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip bad STREAM_TOKENS part %q: %v\n", p, err)
			continue
		}
		out = append(out, int32(n))
	}
	if len(out) == 0 {
		out = make([]int32, len(defaults))
		copy(out, defaults)
	}
	return out
}
