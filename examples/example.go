// Comprehensive REST + optional WebSocket smoke test for the Go Arrow SDK.
//
// Required env (.env): USER_ID, PASSWORD, TOTP_KEY, APP_ID, APP_SECRET
//
// Optional:
//
//	SDK_DEBUG=1
//	TEST_SYMBOL=RELIANCE-EQ          equity symbol for quotes/margin/candles
//	TEST_INDEX=NIFTY                 index underlying for option chain
//	TEST_EXPIRY=16-JUN-2026          option chain expiry (or first from symbols API)
//	PLACE_ORDER=1                    place a far-from-market limit order
//	SKIP_STREAMS=1 / -no-streams     REST only
//	SKIP_HFT=1                       order stream only (no HFT socket)
//	STREAM_DURATION=30s              cap WebSocket run time
//	HFT_SYMBOLS=NSE.SBIN-EQ          comma-separated HFT symbols
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/arrow-trade/go-arrow/arrow"
	"github.com/joho/godotenv"
)

const logFile = "api_test.log"

var (
	testSymbol = "RELIANCE-EQ"
	testIndex  = "NIFTY"
	testExpiry = "16-JUN-2026"
)

type apiResult struct {
	API    string `json:"api"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	godotenv.Load()

	noStreams := flag.Bool("no-streams", false, "skip WebSocket tests")
	streamSec := flag.Int("stream-sec", 0, "optional seconds cap on streams")
	flag.Parse()

	if s := strings.TrimSpace(os.Getenv("TEST_SYMBOL")); s != "" {
		testSymbol = s
	}
	if s := strings.TrimSpace(os.Getenv("TEST_INDEX")); s != "" {
		testIndex = s
	}
	if s := strings.TrimSpace(os.Getenv("TEST_EXPIRY")); s != "" {
		testExpiry = s
	}

	userID := os.Getenv("USER_ID")
	password := os.Getenv("PASSWORD")
	totpKey := os.Getenv("TOTP_KEY")
	appID := os.Getenv("APP_ID")
	appSecret := os.Getenv("APP_SECRET")
	if userID == "" || password == "" || totpKey == "" || appID == "" || appSecret == "" {
		fmt.Println("Set USER_ID, PASSWORD, TOTP_KEY, APP_ID, APP_SECRET (e.g. in .env)")
		os.Exit(1)
	}

	_ = os.WriteFile(logFile, []byte(fmt.Sprintf("API smoke test %s\n", time.Now().Format(time.RFC3339))), 0o644)

	client := arrow.NewClient(appID, appSecret)
	if truthy(os.Getenv("SDK_DEBUG")) {
		client.SetDebug(true)
	}

	var results []apiResult

	fmt.Printf("Logging in as user=%s appID=%s\n", userID, appID)
	if err := client.AutoLogin(userID, password, totpKey); err != nil {
		fmt.Println("AutoLogin failed:", err)
		os.Exit(1)
	}
	fmt.Println("Login OK, token length:", len(client.GetToken()))

	// --- User & portfolio ---
	runAPI("GetUserDetails", false, &results, func() (any, error) {
		return client.GetUserDetails()
	})
	runAPI("GetHoldings", false, &results, func() (any, error) {
		return client.GetHoldings()
	})
	runAPI("GetPositions", false, &results, func() (any, error) {
		return client.GetPositions()
	})
	runAPI("GetLimits", false, &results, func() (any, error) {
		return client.GetLimits()
	})
	runAPI("GetOrderBook", false, &results, func() (any, error) {
		return client.GetOrderBook()
	})
	runAPI("GetTradeBook", false, &results, func() (any, error) {
		return client.GetTradeBook()
	})

	// --- Margin ---
	runAPI("GetMargin", false, &results, func() (any, error) {
		return client.GetMargin(arrow.MarginRequest{
			Exchange:         arrow.ExchangeNSE,
			Symbol:           testSymbol,
			Quantity:         "1",
			Price:            "250000",
			Product:          arrow.ProductCNC,
			TransactionType:  arrow.TransactionTypeBuy,
			Order:            arrow.OrderTypeLimit,
			IncludePositions: false,
		})
	})
	runAPI("GetBasketMargin", false, &results, func() (any, error) {
		return client.GetBasketMargin(arrow.BasketMarginRequest{
			Orders: []arrow.MarginRequest{{
				Exchange:         arrow.ExchangeNFO,
				Symbol:           testSymbol,
				Quantity:         "1",
				Price:            "250000",
				Product:          arrow.ProductCNC,
				TransactionType:  arrow.TransactionTypeBuy,
				Order:            arrow.OrderTypeLimit,
				IncludePositions: false,
			}},
			IncludePositions: false,
		})
	})

	// --- Market data ---
	var eqToken string
	runAPI("GetQuote (LTP)", false, &results, func() (any, error) {
		q, err := client.GetQuote(arrow.ExchangeNSE, testSymbol, arrow.InfoQuoteLTP)
		if err != nil {
			return nil, err
		}
		if t, ok := q["token"]; ok {
			eqToken = fmt.Sprint(t)
		}
		return q, nil
	})
	runAPI("GetQuotes (batch LTP)", false, &results, func() (any, error) {
		return client.GetQuotes([]arrow.QuoteInstrument{
			{Exchange: string(arrow.ExchangeNSE), Symbol: testSymbol},
		}, arrow.InfoQuoteLTP)
	})
	runAPI("GetQuote (INDEX)", false, &results, func() (any, error) {
		return client.GetQuote(arrow.ExchangeINDEX, testIndex, arrow.InfoQuoteLTP)
	})

	var ocSymbols arrow.OptionChainSymbolsByCategory
	runAPI("GetAllOptionChainSymbols", false, &results, func() (any, error) {
		syms, err := client.GetAllOptionChainSymbols()
		if err != nil {
			return nil, err
		}
		ocSymbols = syms
		return syms, nil
	})

	expiry := testExpiry
	if exs, ok := ocSymbols["indices"]["INDEX:"+testIndex]; ok && len(exs) > 0 {
		expiry = exs[0]
	}
	runAPI("GetOptionChain", false, &results, func() (any, error) {
		raw, err := client.GetOptionChain(arrow.OptionChainRequest{
			Underlying: testIndex,
			Exchange:   arrow.ExchangeINDEX,
			Count:      "5",
			Expiry:     expiry,
		})
		if err != nil {
			return nil, err
		}
		return json.RawMessage(raw), nil
	})

	runAPI("GetHolidays", false, &results, func() (any, error) {
		return client.GetHolidays()
	})
	runAPI("GetIndexList", false, &results, func() (any, error) {
		return client.GetIndexList()
	})
	runAPI("GetInstruments (nse, first 3 rows)", false, &results, func() (any, error) {
		rows, err := client.GetInstruments(arrow.InstrumentSegmentNSE)
		if err != nil {
			return nil, err
		}
		n := 3
		if len(rows) < n {
			n = len(rows)
		}
		return rows[:n], nil
	})

	toT := time.Now()
	fromT := toT.Add(-14 * 24 * time.Hour)
	fromTS := fromT.Format("2006-01-02T15:04:05")
	toTS := toT.Format("2006-01-02T15:04:05")
	if eqToken == "" {
		eqToken = "3045"
	}
	runAPI("GetCandleData (equity)", false, &results, func() (any, error) {
		raw, err := client.GetCandleData(arrow.ExchangeNSE, eqToken, "day", fromTS, toTS, false)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(raw), nil
	})

	runAPI("GetGreeks", true, &results, func() (any, error) {
		tok, _ := strconv.Atoi(eqToken)
		if tok == 0 {
			tok = 3045
		}
		raw, err := client.GetGreeks([]int{tok})
		if err != nil {
			return nil, err
		}
		return json.RawMessage(raw), nil
	})

	// --- Order lifecycle (optional) ---
	var standingOrderID string
	if truthy(os.Getenv("PLACE_ORDER")) {
		runAPI("PlaceOrder", false, &results, func() (any, error) {
			resp, err := placeTestOrder(client)
			if err != nil {
				return nil, err
			}
			standingOrderID = resp.Data.OrderNo
			return resp, nil
		})
	} else {
		fmt.Println("\nSkipping PlaceOrder (set PLACE_ORDER=1 to test order placement).")
	}

	orders, _ := client.GetOrderBook()
	for _, o := range orders {
		if isStanding(o.OrderStatus) {
			standingOrderID = o.ID
			break
		}
	}
	if standingOrderID != "" {
		runAPI("GetOrder", false, &results, func() (any, error) {
			return client.GetOrder(standingOrderID)
		})
		if truthy(os.Getenv("PLACE_ORDER")) {
			// runAPI("ModifyOrder", true, &results, func() (any, error) {
			// 	return modifyTestOrder(client, standingOrderID)
			// })
			// runAPI("CancelOrder", true, &results, func() (any, error) {
			// 	return nil, client.CancelOrder("regular", standingOrderID)
			// })
		}
	} else {
		fmt.Println("\nNo standing order for GetOrder / ModifyOrder / CancelOrder tests.")
	}

	printSummary(results)

	skipStreams := *noStreams || truthy(os.Getenv("SKIP_STREAMS"))
	if skipStreams {
		fmt.Println("\nSkipping WebSockets (unset SKIP_STREAMS / omit -no-streams to test streams).")
		return
	}
	runStreams(client, *streamSec)
}

func runAPI(name string, optional bool, results *[]apiResult, fn func() (any, error)) {
	fmt.Printf("\n%s %s\n%s\n", strings.Repeat("=", 60), name, strings.Repeat("=", 60))
	v, err := fn()
	entry := apiResult{API: name}
	if err != nil {
		if optional {
			entry.Status = "SKIP"
			entry.Error = err.Error()
			fmt.Printf("⊘ SKIP: %v\n", err)
		} else {
			entry.Status = "FAIL"
			entry.Error = err.Error()
			fmt.Printf("✗ FAIL: %v\n", err)
		}
	} else {
		entry.Status = "OK"
		entry.Detail = truncateJSON(v, 600)
		fmt.Printf("✓ OK\n%s\n", entry.Detail)
	}
	*results = append(*results, entry)
	appendLog(entry)
}

func appendLog(entry apiResult) {
	line, _ := json.Marshal(entry)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}

func printSummary(results []apiResult) {
	ok, fail, skip := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case "OK":
			ok++
		case "FAIL":
			fail++
		case "SKIP":
			skip++
		}
	}
	fmt.Printf("\n%s\nSUMMARY: %d passed, %d failed, %d skipped (of %d)\n%s\n",
		strings.Repeat("=", 60), ok, fail, skip, len(results), strings.Repeat("=", 60))
	for _, r := range results {
		fmt.Printf("  [%s] %s\n", r.Status, r.API)
	}
}

func truncateJSON(v any, max int) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		b = []byte(fmt.Sprint(v))
	}
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + fmt.Sprintf("\n... (%d more chars)", len(b)-max)
}

func truthy(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "1" || s == "true" || s == "yes"
}

func isStanding(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "OPEN", "TRIGGER_PENDING", "PARTIALLY_FILLED", "PENDING":
		return true
	default:
		return false
	}
}

func safeLimitPrice(client *arrow.Client, symbol string) string {
	q, err := client.GetQuote(arrow.ExchangeNSE, symbol, arrow.InfoQuoteLTP)
	if err != nil {
		return "100"
	}
	ltp, _ := strconv.ParseFloat(fmt.Sprint(q["ltp"]), 64)
	if ltp > 10_000 {
		ltp /= 100
	}
	price := ltp * 0.5
	if price < 1 {
		price = 1
	}
	return fmt.Sprintf("%.0f", price*100)
}

func placeTestOrder(client *arrow.Client) (*arrow.OrderResponse, error) {
	symbol := testSymbol
	if s := strings.TrimSpace(os.Getenv("TEST_ORDER_SYMBOL")); s != "" {
		symbol = s
	}
	quantity := "1"
	if q := strings.TrimSpace(os.Getenv("TEST_ORDER_QUANTITY")); q != "" {
		quantity = q
	}
	order := arrow.OrderRequest{
		Exchange:         string(arrow.ExchangeNSE),
		Symbol:           symbol,
		Quantity:         quantity,
		Product:          string(arrow.ProductCNC),
		TransactionType:  string(arrow.TransactionTypeBuy),
		OrderType:        string(arrow.OrderTypeLimit),
		Price:            safeLimitPrice(client, symbol),
		Validity:         string(arrow.ValidityDAY),
		MarketProtection: false,
	}
	fmt.Printf("PlaceOrder: %+v\n", order)
	return client.PlaceOrder("regular", order)
}

func modifyTestOrder(client *arrow.Client, orderID string) (*arrow.OrderResponse, error) {
	order := arrow.OrderRequest{
		Exchange:         string(arrow.ExchangeNFO),
		Symbol:           testSymbol,
		Quantity:         "65",
		Product:          string(arrow.ProductNRML),
		TransactionType:  string(arrow.TransactionTypeBuy),
		OrderType:        string(arrow.OrderTypeLimit),
		Price:            "605",
		Validity:         string(arrow.ValidityDAY),
		MarketProtection: false,
	}
	return client.ModifyOrder("regular", orderID, order)
}

func runStreams(client *arrow.Client, streamSec int) {
	var dur time.Duration
	if streamSec > 0 {
		dur = time.Duration(streamSec) * time.Second
	} else if d := strings.TrimSpace(os.Getenv("STREAM_DURATION")); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			dur = parsed
		}
	}

	logPath := strings.TrimSpace(os.Getenv("HFT_LOG_FILE"))
	if logPath == "" {
		logPath = "hft-stream.log"
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Println("open stream log:", err)
		return
	}
	defer logFile.Close()
	streamLog := log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags|log.Lmicroseconds)

	skipHFT := truthy(os.Getenv("SKIP_HFT"))
	var streams *arrow.ArrowStreams
	var dsOnly *arrow.DataStream
	if skipHFT {
		streams, err = client.NewStreams()
	} else {
		streams, err = client.NewStreamsWithHFT()
		if err != nil {
			streamLog.Printf("HFT unavailable, falling back to order + ds: %v", err)
			streams, err = client.NewStreams()
		} else if ds, dsErr := client.ConnectDataStream(); dsErr == nil {
			dsOnly = ds
		} else {
			streamLog.Printf("ds stream unavailable: %v", dsErr)
		}
	}
	if err != nil {
		streamLog.Printf("streams connect error: %v", err)
		return
	}
	defer streams.Close()
	if dsOnly != nil {
		defer dsOnly.Close()
	}

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var ctx context.Context
	var cancel context.CancelFunc
	if dur > 0 {
		ctx, cancel = context.WithTimeout(sigCtx, dur)
	} else {
		ctx = sigCtx
		cancel = func() {}
	}
	defer cancel()

	streamLog.Println("WebSocket smoke test started (order updates + optional HFT + ds LTP)")

	go streams.OrderStream.ReadUpdates(ctx, func(update map[string]any) {
		streamLog.Printf("order update: %+v", update)
	}, func(err error) {
		streamLog.Printf("order stream ended: %v", err)
	})

	// Standard ds.arrow.trade token stream (LTP subscribe).
	ds := streams.DataStream
	if ds == nil {
		ds = dsOnly
	}
	if ds != nil {
		if tok := instrumentTokenFromEnv(); tok > 0 {
			if err := ds.Subscribe(arrow.StreamModeLTP, []int32{tok}); err != nil {
				streamLog.Printf("ds subscribe error: %v", err)
			} else {
				streamLog.Printf("ds subscribed LTP token=%d", tok)
				go ds.ReadTicks(ctx, func(tick arrow.MarketTick) {
					streamLog.Printf("ds tick: token=%d ltp=%d mode=%s", tick.Token, tick.LTP, tick.Mode)
				}, func(err error) {
					streamLog.Printf("ds stream ended: %v", err)
				})
			}
		}
	}

	if streams.HFTDataStream != nil {
		hftSyms := parseHFTSymbols(os.Getenv("HFT_SYMBOLS"), []string{"NSE.SBIN-EQ"})
		latency := 50
		if ls := strings.TrimSpace(os.Getenv("HFT_LATENCY_MS")); ls != "" {
			if n, e := strconv.Atoi(ls); e == nil && n > 0 {
				latency = n
			}
		}
		if err := streams.HFTDataStream.SubscribeHFTSymbols("ltpc", hftSyms, latency); err != nil {
			streamLog.Printf("HFT subscribe error: %v", err)
		} else {
			streamLog.Printf("HFT subscribed symbols=%v latency_ms=%d", hftSyms, latency)
			go streams.HFTDataStream.ReadHFT(ctx,
				func(t arrow.HFTLTPTick) {
					streamLog.Printf("HFT LTP: token=%d ltp=%d", t.Token, t.LTP)
				},
				func(t arrow.HFTFullTick) {
					streamLog.Printf("HFT full: token=%d ltp=%d", t.Token, t.LTP)
				},
				func(r arrow.HFTResponsePacket) {
					streamLog.Printf("HFT response: code=%q ok=%d", r.ErrorCode, r.SuccessCount)
				},
				func(err error) {
					streamLog.Printf("HFT stream ended: %v", err)
				},
			)
		}
	}

	<-ctx.Done()
	streamLog.Printf("streams stopped: %v", ctx.Err())
}

func instrumentTokenFromEnv() int32 {
	if s := strings.TrimSpace(os.Getenv("TEST_CANDLE_EQ_TOKEN")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return int32(n)
		}
	}
	return 3045
}

func parseHFTSymbols(raw string, defaults []string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		out := make([]string, len(defaults))
		copy(out, defaults)
		return out
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = make([]string, len(defaults))
		copy(out, defaults)
	}
	return out
}
