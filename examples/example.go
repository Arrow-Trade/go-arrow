package main

import (
	"bytes"
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

// Env / flags for local testing:
//   USER_ID, PASSWORD, TOTP_KEY, APP_ID, APP_SECRET — required for AutoLogin
//   SDK_DEBUG=1 — enable verbose SDK logs
//   SKIP_STREAMS=1 — only run REST calls (no WebSockets)
//   SKIP_HFT=1 — order-updates WebSocket only (no HFT socket)
//   HFT_SYMBOLS — comma-separated HFT symbol names for SubscribeHFTSymbols (default NSE.SBIN-EQ). See https://docs.arrow.trade/python-sdk/websocket-streaming/
//   HFT_LATENCY_MS — tick throttle for HFT subscribe (default 50; range 50–60000 per docs)
//   HFT_LOG_FILE — append stream + HFT lines here as well as stdout (default hft-stream.log)
//   STREAM_DURATION — optional cap, e.g. 30s, 1m; if unset and -stream-sec=0, streams run until SIGINT/SIGTERM
//   TEST_QUOTE_EXCHANGE + TEST_QUOTE_SYMBOL — if both set, calls REST GetQuote (InfoQuoteLTP) after login
//   TEST_OPTIONCHAIN_UNDERLYING, TEST_OPTIONCHAIN_EXCHANGE, TEST_OPTIONCHAIN_COUNT, TEST_OPTIONCHAIN_EXPIRY —
//     if all four set, calls REST GetOptionChain; otherwise, if GetAllOptionChainSymbols returns INDEX:NIFTY expiries,
//     uses underlying NIFTY, exchange NFO, count 5, and the first listed expiry as defaults.
//   TEST_CANDLE_EQ_TOKEN, TEST_CANDLE_FUT_TOKEN — instrument tokens for GetCandleData (defaults: 3045 NSE equity doc example, 41927 NFO doc example).
//   TEST_CANDLE_INTERVAL — candle interval (default day); see https://docs.arrow.trade/rest-api/historical-candle-data/
//   PLACE_ORDER=1 — call REST PlaceOrder (off by default). Optional TEST_ORDER_* override exchange, symbol, qty, product, txn, type, price, validity, disclosed qty, remarks, variety (default regular), mpp.

func main() {
	godotenv.Load()

	noStreams := flag.Bool("no-streams", false, "skip order + market WebSocket test")
	streamSec := flag.Int("stream-sec", 0, "optional seconds cap on streams (0 = until SIGINT, or use STREAM_DURATION)")
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

	if os.Getenv("PLACE_ORDER") == "1" || strings.EqualFold(os.Getenv("PLACE_ORDER"), "true") {
		resp, perr := placeOrder(client)
		if perr != nil {
			fmt.Println("PlaceOrder error:", perr)
		} else {
			fmt.Printf("PlaceOrder: orderNo=%s requestTime=%s\n", resp.Data.OrderNo, resp.Data.RequestTime)
		}
	} else {
		fmt.Println("Skipping PlaceOrder (set PLACE_ORDER=1 to place a test order; optional TEST_ORDER_* env vars).")
	}

	ocSymbols, err := client.GetAllOptionChainSymbols()
	if err != nil {
		fmt.Println("GetAllOptionChainSymbols error:", err)
	} else {
		fmt.Printf("Option chain symbols: %+v\n", ocSymbols)
	}

	ocUnderlying := strings.TrimSpace(os.Getenv("TEST_OPTIONCHAIN_UNDERLYING"))
	ocExchange := strings.TrimSpace(os.Getenv("TEST_OPTIONCHAIN_EXCHANGE"))
	ocCount := strings.TrimSpace(os.Getenv("TEST_OPTIONCHAIN_COUNT"))
	ocExpiry := strings.TrimSpace(os.Getenv("TEST_OPTIONCHAIN_EXPIRY"))
	if ocUnderlying == "" || ocExchange == "" || ocCount == "" || ocExpiry == "" {
		if exs, ok := ocSymbols["indices"]["INDEX:NIFTY"]; ok && len(exs) > 0 {
			ocUnderlying = "NIFTY"
			ocExchange = string(arrow.ExchangeNFO)
			ocCount = "5"
			ocExpiry = exs[0]
			fmt.Printf("Option chain request defaults from symbols: underlying=%s exchange=%s count=%s expiry=%s\n",
				ocUnderlying, ocExchange, ocCount, ocExpiry)
		}
	}
	if ocUnderlying != "" && ocExchange != "" && ocCount != "" && ocExpiry != "" {
		chain, ocErr := client.GetOptionChain(arrow.OptionChainRequest{
			Underlying: ocUnderlying,
			Exchange:   arrow.ExchangeINDEX,
			Count:      ocCount,
			Expiry:     ocExpiry,
		})
		if ocErr != nil {
			fmt.Println("GetOptionChain error:", ocErr)
		} else {
			var buf bytes.Buffer
			if indentErr := json.Indent(&buf, chain, "", "  "); indentErr != nil {
				fmt.Printf("Option chain (raw): %s\n", string(chain))
			} else {
				fmt.Printf("Option chain:\n%s\n", buf.String())
			}
		}
	} else {
		fmt.Println("Skipping GetOptionChain (set all of TEST_OPTIONCHAIN_UNDERLYING, TEST_OPTIONCHAIN_EXCHANGE, TEST_OPTIONCHAIN_COUNT, TEST_OPTIONCHAIN_EXPIRY, or rely on INDEX:NIFTY in option-chain symbols).")
	}

	quote, qerr := client.GetQuote(arrow.ExchangeINDEX, "SENSEX", arrow.InfoQuoteLTP)
	if qerr != nil {
		fmt.Println("GetQuote error:", qerr)
	} else {
		fmt.Printf("REST quote (INDEX NIFTY): %+v\n", quote)
	}

	// Historical OHLCV: NSE equity (Eq) and NFO futures — tokens are instrument IDs (see instruments CSV / broker docs).
	eqCandleTok := strings.TrimSpace(os.Getenv("TEST_CANDLE_EQ_TOKEN"))
	if eqCandleTok == "" {
		eqCandleTok = "3045" // SBIN example from Arrow historical candle docs
	}
	futCandleTok := strings.TrimSpace(os.Getenv("TEST_CANDLE_FUT_TOKEN"))
	if futCandleTok == "" {
		futCandleTok = "41927" // NFO example token from docs (rolls with contract; set TEST_CANDLE_FUT_TOKEN for a live future)
	}
	candleInterval := strings.TrimSpace(os.Getenv("TEST_CANDLE_INTERVAL"))
	if candleInterval == "" {
		candleInterval = "day"
	}
	toT := time.Now()
	fromT := toT.Add(-14 * 24 * time.Hour)
	fromTS := fromT.Format("2006-01-02T15:04:05")
	toTS := toT.Format("2006-01-02T15:04:05")

	candlesEQ, cerr := client.GetCandleData(arrow.ExchangeNSE, eqCandleTok, candleInterval, fromTS, toTS, false)
	if cerr != nil {
		fmt.Println("GetCandleData (NSE equity) error:", cerr)
	} else {
		var buf bytes.Buffer
		if indentErr := json.Indent(&buf, candlesEQ, "", "  "); indentErr != nil {
			fmt.Printf("Candles NSE EQ (raw): %s\n", string(candlesEQ))
		} else {
			fmt.Printf("Candles NSE EQ (token=%s interval=%s):\n%s\n", eqCandleTok, candleInterval, buf.String())
		}
	}

	candlesFut, ferr := client.GetCandleData(arrow.ExchangeNFO, futCandleTok, candleInterval, fromTS, toTS, true)
	if ferr != nil {
		fmt.Println("GetCandleData (NFO futures, oi=1) error:", ferr)
	} else {
		var buf bytes.Buffer
		if indentErr := json.Indent(&buf, candlesFut, "", "  "); indentErr != nil {
			fmt.Printf("Candles NFO (raw): %s\n", string(candlesFut))
		} else {
			fmt.Printf("Candles NFO futures (token=%s interval=%s, open interest):\n%s\n", futCandleTok, candleInterval, buf.String())
		}
	}

	skipStreams := *noStreams || os.Getenv("SKIP_STREAMS") == "1" || strings.EqualFold(os.Getenv("SKIP_STREAMS"), "true")
	if skipStreams {
		fmt.Println("Skipping WebSockets (use default or unset SKIP_STREAMS / omit -no-streams to test streams).")
		return
	}

	var dur time.Duration
	if *streamSec > 0 {
		dur = time.Duration(*streamSec) * time.Second
	} else if d := strings.TrimSpace(os.Getenv("STREAM_DURATION")); d != "" {
		if parsed, perr := time.ParseDuration(d); perr == nil {
			dur = parsed
		}
	}

	logPath := strings.TrimSpace(os.Getenv("HFT_LOG_FILE"))
	if logPath == "" {
		logPath = "hft-stream.log"
	}
	logFile, lerr := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if lerr != nil {
		fmt.Println("open log file:", lerr)
		return
	}
	defer logFile.Close()
	streamLog := log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags|log.Lmicroseconds)

	skipHFT := os.Getenv("SKIP_HFT") == "1" || strings.EqualFold(os.Getenv("SKIP_HFT"), "true")
	if dur > 0 {
		streamLog.Printf("connecting order (+ HFT unless SKIP_HFT); max duration %v; also logging to %s", dur, logPath)
	} else {
		streamLog.Printf("connecting order (+ HFT unless SKIP_HFT); run until SIGINT/SIGTERM (no duration cap); also logging to %s", logPath)
	}

	var streams *arrow.ArrowStreams
	if skipHFT {
		streams, err = client.NewStreamsOrderOnly()
	} else {
		streams, err = client.NewStreamsWithHFT()
		if err != nil {
			streamLog.Printf("HFT socket unavailable, order stream only: %v", err)
			streams, err = client.NewStreamsOrderOnly()
		}
	}
	if err != nil {
		streamLog.Printf("streams connect error: %v", err)
		return
	}
	defer streams.Close()

	sigCtx, stopSig := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSig()

	var ctx context.Context
	var cancel context.CancelFunc
	if dur > 0 {
		ctx, cancel = context.WithTimeout(sigCtx, dur)
		defer cancel()
	} else {
		ctx = sigCtx
		cancel = func() {}
	}

	go streams.OrderStream.ReadUpdates(ctx, func(update map[string]any) {
		streamLog.Printf("order update: %+v", update)
	}, func(err error) {
		streamLog.Printf("ORDER stream read ended (disconnect or error): %v", err)
	})

	if streams.HFTDataStream != nil {
		hftSyms := parseHFTSymbols(os.Getenv("HFT_SYMBOLS"), []string{"NSE.SBIN-EQ"})
		hftLatency := 50
		if ls := strings.TrimSpace(os.Getenv("HFT_LATENCY_MS")); ls != "" {
			if n, e := strconv.Atoi(ls); e == nil && n > 0 {
				hftLatency = n
			}
		}
		if err := streams.HFTDataStream.SubscribeHFTSymbols("ltpc", hftSyms, hftLatency); err != nil {
			streamLog.Printf("HFT subscribe error: %v", err)
		} else {
			streamLog.Printf("HFT subscribed ltpc symbols=%v latency_ms=%d (prices on wire are in paise)", hftSyms, hftLatency)
			go streams.HFTDataStream.ReadHFT(ctx,
				func(t arrow.HFTLTPTick) {
					streamLog.Printf("HFT LTP: token=%d ltp_paise=%d vol=%d exch_seg=%d", t.Token, t.LTP, t.Volume, t.ExchSeg)
				},
				func(t arrow.HFTFullTick) {
					streamLog.Printf("HFT full: token=%d ltp_paise=%d bid0_paise=%d ask0_paise=%d", t.Token, t.LTP, t.BidPx[0], t.AskPx[0])
				},
				func(r arrow.HFTResponsePacket) {
					streamLog.Printf("HFT response: code=%q msg=%q req=%s mode=%s ok=%d err=%d",
						r.ErrorCode, r.ErrorMsg, r.RequestTypeStr, r.ModeStr, r.SuccessCount, r.ErrorCount)
				},
				func(err error) {
					streamLog.Printf("HFT stream read ended (disconnect or error): %v", err)
				},
			)
		}
	}

	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		streamLog.Printf("stream run stopped: duration limit reached (%v)", dur)
	} else {
		streamLog.Printf("stream run stopped: %v", ctx.Err())
	}
}

// placeOrder submits a regular variety order using TEST_ORDER_* env vars (see file header).
func placeOrder(client *arrow.Client) (*arrow.OrderResponse, error) {
	exchange := strings.TrimSpace(os.Getenv("TEST_ORDER_EXCHANGE"))
	if exchange == "" {
		exchange = string(arrow.ExchangeNSE)
	}
	symbol := strings.TrimSpace(os.Getenv("TEST_ORDER_SYMBOL"))
	if symbol == "" {
		symbol = "YESBANK-EQ"
	}
	quantity := strings.TrimSpace(os.Getenv("TEST_ORDER_QUANTITY"))
	if quantity == "" {
		quantity = "1"
	}
	product := strings.TrimSpace(os.Getenv("TEST_ORDER_PRODUCT"))
	if product == "" {
		product = string(arrow.ProductCNC)
	}
	txn := strings.TrimSpace(os.Getenv("TEST_ORDER_TRANSACTION"))
	if txn == "" {
		txn = string(arrow.TransactionTypeBuy)
	}
	orderKind := strings.TrimSpace(os.Getenv("TEST_ORDER_TYPE"))
	if orderKind == "" {
		orderKind = string(arrow.OrderTypeLimit)
	}
	price := strings.TrimSpace(os.Getenv("TEST_ORDER_PRICE"))
	if price == "" {
		price = "1" // far from market; override TEST_ORDER_PRICE for a real limit
	}
	validity := strings.TrimSpace(os.Getenv("TEST_ORDER_VALIDITY"))
	if validity == "" {
		validity = string(arrow.ValidityDAY)
	}
	variety := strings.TrimSpace(os.Getenv("TEST_ORDER_VARIETY"))
	if variety == "" {
		variety = "regular"
	}

	order := arrow.OrderRequest{
		Exchange:         exchange,
		Symbol:           symbol,
		Quantity:         quantity,
		Product:          product,
		TransactionType:  txn,
		OrderType:        orderKind,
		Price:            price,
		Validity:         validity,
		MarketProtection: os.Getenv("TEST_ORDER_MPP") == "1" || strings.EqualFold(os.Getenv("TEST_ORDER_MPP"), "true"),
	}

	fmt.Printf("PlaceOrder request: %+v\n", order)
	if dq := strings.TrimSpace(os.Getenv("TEST_ORDER_DISCLOSED_QTY")); dq != "" {
		order.DisclosedQty = dq
	}
	if remarks := strings.TrimSpace(os.Getenv("TEST_ORDER_REMARKS")); remarks != "" {
		order.Remarks = remarks
	}

	fmt.Printf("PlaceOrder request: variety=%s exchange=%s symbol=%s qty=%s product=%s txn=%s type=%s price=%s validity=%s mpp=%v\n",
		variety, order.Exchange, order.Symbol, order.Quantity, order.Product,
		order.TransactionType, order.OrderType, order.Price, order.Validity, order.MarketProtection)

	return client.PlaceOrder(variety, order)
}

func parseHFTSymbols(raw string, defaults []string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		out := make([]string, len(defaults))
		copy(out, defaults)
		return out
	}
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = make([]string, len(defaults))
		copy(out, defaults)
	}
	return out
}
