package arrow

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	orderStreamURL = "wss://order-updates.arrow.trade"
	dataStreamURL  = "wss://ds.arrow.trade"
)

// StreamMode is the subscription mode for the token-based market WebSocket (wss://ds.arrow.trade).
// Inbound ticks are binary, big-endian int fields. Base lengths: 13 (ltp) / 17 (ltpc) / 93 (quote) /
// 241|249 (full). During Closing Auction Session (CAS, after ~15:15 IST) every mode appends a
// 16-byte trailer → 29 / 33 / 109 / 265. REST /info/quote uses InfoQuoteMode in quote.go.
type StreamMode string

const (
	StreamModeLTP   StreamMode = "ltp"
	StreamModeLTPC  StreamMode = "ltpc"
	StreamModeQuote StreamMode = "quote"
	StreamModeFull  StreamMode = "full"

	casTrailerLen = 16
)

type DepthLevel struct {
	Quantity int64 `json:"quantity"`
	Price    int32 `json:"price"`
	Orders   int16 `json:"orders"`
}

type MarketTick struct {
	Token             int32        `json:"token"`
	Mode              StreamMode   `json:"mode"`
	LTP               int32        `json:"ltp"`
	Close             int32        `json:"close"`
	NetChange         float64      `json:"netChange"`
	ChangeFlag        int8         `json:"changeFlag"`
	LTQ               int32        `json:"ltq"`
	AvgPrice          int32        `json:"avgPrice"`
	TotalBuyQuantity  int64        `json:"totalBuyQuantity"`
	TotalSellQuantity int64        `json:"totalSellQuantity"`
	Open              int32        `json:"open"`
	High              int32        `json:"high"`
	Low               int32        `json:"low"`
	Volume            int64        `json:"volume"`
	LTT               int32        `json:"ltt"`
	Time              int32        `json:"time"`
	OI                int64        `json:"oi"`
	OIDayHigh         int64        `json:"oiDayHigh"`
	OIDayLow          int64        `json:"oiDayLow"`
	LowerLimit        int32        `json:"lowerLimit"`
	UpperLimit        int32        `json:"upperLimit"`
	Bids              []DepthLevel `json:"bids"`
	Asks              []DepthLevel `json:"asks"`
	// CAS fields (appended to every mode after ~15:15 IST: +16 bytes).
	// ImbalanceQty is signed (negative = sell-side imbalance).
	ImbalanceQty    int64 `json:"imbalanceQty"`
	IndicativeClose int32 `json:"indicativeClose"`
	RefPrice        int32 `json:"refPrice"`
}

type DataStream struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *Client) ConnectDataStream() (*DataStream, error) {
	q := url.Values{}
	q.Set("appID", c.Config.AppID)
	q.Set("token", c.Config.Token)
	u := fmt.Sprintf("%s?%s", dataStreamURL, q.Encode())
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return nil, err
	}
	return &DataStream{conn: conn}, nil
}

func (s *DataStream) Close() error {
	return s.conn.Close()
}

func (s *DataStream) Subscribe(mode StreamMode, tokens []int32) error {
	return s.sendSubMessage("sub", mode, tokens)
}

func (s *DataStream) Unsubscribe(mode StreamMode, tokens []int32) error {
	return s.sendSubMessage("unsub", mode, tokens)
}

func (s *DataStream) sendSubMessage(code string, mode StreamMode, tokens []int32) error {
	msg := map[string]any{
		"code":       code,
		"mode":       mode,
		string(mode): tokens,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteJSON(msg)
}

func (s *DataStream) ReadTicks(ctx context.Context, onTick func(MarketTick), onError func(error)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, payload, err := s.conn.ReadMessage()
		if err != nil {
			if onError != nil && !errors.Is(err, websocket.ErrCloseSent) {
				onError(err)
			}
			return
		}
		if len(payload) < 13 {
			// Heartbeats / control payloads (e.g. 1 byte) — ignore.
			continue
		}
		tick, err := ParseMarketTick(payload)
		if err != nil {
			if onError != nil {
				onError(err)
			}
			continue
		}
		onTick(tick)
	}
}

func ParseMarketTick(data []byte) (MarketTick, error) {
	switch len(data) {
	case 13:
		return parseLTP(data), nil
	case 29: // LTP + CAS trailer
		tick := parseLTP(data)
		applyCAS(&tick, data, 13)
		return tick, nil
	case 17:
		return parseLTPC(data), nil
	case 33: // LTPC + CAS trailer
		tick := parseLTPC(data)
		applyCAS(&tick, data, 17)
		return tick, nil
	case 93:
		return parseQuote(data), nil
	case 109: // Quote + CAS trailer
		tick := parseQuote(data)
		applyCAS(&tick, data, 93)
		return tick, nil
	case 241, 249:
		return parseFull(data), nil
	case 265: // Full + CAS trailer
		tick := parseFull(data)
		applyCAS(&tick, data, 249)
		return tick, nil
	default:
		return MarketTick{}, fmt.Errorf("unsupported market tick payload size: %d", len(data))
	}
}

func applyCAS(tick *MarketTick, data []byte, off int) {
	if len(data) < off+casTrailerLen {
		return
	}
	tick.ImbalanceQty = beI64(data[off : off+8])
	tick.IndicativeClose = beI32(data[off+8 : off+12])
	tick.RefPrice = beI32(data[off+12 : off+16])
}

func parseLTP(data []byte) MarketTick {
	return MarketTick{
		Token: beI32(data[0:4]),
		LTP:   beI32(data[4:8]),
		Mode:  StreamModeLTP,
	}
}

func parseLTPC(data []byte) MarketTick {
	ltp := beI32(data[4:8])
	closePx := beI32(data[13:17])
	tick := MarketTick{
		Token:      beI32(data[0:4]),
		LTP:        ltp,
		Close:      closePx,
		Mode:       StreamModeLTPC,
		ChangeFlag: int8(data[8]),
	}
	if closePx != 0 {
		tick.NetChange = float64(ltp-closePx) * 100 / float64(closePx)
	}
	return tick
}

func parseQuote(data []byte) MarketTick {
	tick := parseLTPC(data)
	tick.Mode = StreamModeQuote
	tick.LTQ = beI32(data[13:17])
	tick.AvgPrice = beI32(data[17:21])
	tick.TotalBuyQuantity = beI64(data[21:29])
	tick.TotalSellQuantity = beI64(data[29:37])
	tick.Open = beI32(data[37:41])
	tick.High = beI32(data[41:45])
	tick.Close = beI32(data[45:49])
	tick.Low = beI32(data[49:53])
	tick.Volume = beI64(data[53:61])
	tick.LTT = beI32(data[61:65])
	tick.Time = beI32(data[65:69])
	tick.OI = beI64(data[69:77])
	tick.OIDayHigh = beI64(data[77:85])
	tick.OIDayLow = beI64(data[85:93])
	if tick.Close != 0 {
		tick.NetChange = float64(tick.LTP-tick.Close) * 100 / float64(tick.Close)
	}
	return tick
}

func parseFull(data []byte) MarketTick {
	tick := parseQuote(data)
	tick.Mode = StreamModeFull
	tick.LowerLimit = beI32(data[93:97])
	tick.UpperLimit = beI32(data[97:101])
	// 249/265-byte packets reserve 8 bytes before depth; 241-byte legacy starts depth at 101.
	depthOffset := 101
	if len(data) >= 249 {
		depthOffset = 109
	}
	tick.Bids = make([]DepthLevel, 0, 5)
	tick.Asks = make([]DepthLevel, 0, 5)
	for i := 0; i < 10; i++ {
		offset := depthOffset + i*14
		level := DepthLevel{
			Quantity: beI64(data[offset : offset+8]),
			Price:    beI32(data[offset+8 : offset+12]),
			Orders:   int16(binary.BigEndian.Uint16(data[offset+12 : offset+14])),
		}
		if i < 5 {
			tick.Bids = append(tick.Bids, level)
		} else {
			tick.Asks = append(tick.Asks, level)
		}
	}
	return tick
}

type OrderStream struct {
	conn *websocket.Conn
}

func (c *Client) ConnectOrderStream() (*OrderStream, error) {
	q := url.Values{}
	q.Set("appID", c.Config.AppID)
	q.Set("token", c.Config.Token)
	u := fmt.Sprintf("%s?%s", orderStreamURL, q.Encode())
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return nil, err
	}
	return &OrderStream{conn: conn}, nil
}

func (s *OrderStream) Close() error {
	return s.conn.Close()
}

func (s *OrderStream) ReadUpdates(ctx context.Context, onUpdate func(map[string]any), onError func(error)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		mt, payload, err := s.conn.ReadMessage()
		if err != nil {
			if onError != nil && !errors.Is(err, websocket.ErrCloseSent) {
				onError(err)
			}
			return
		}
		if mt != websocket.TextMessage {
			continue
		}
		payload = trimNulls(payload)
		if len(payload) == 0 {
			continue
		}
		var update map[string]any
		if err := json.Unmarshal(payload, &update); err != nil {
			// Non-JSON text (e.g. keepalive); skip without spamming onError.
			continue
		}
		onUpdate(update)
	}
}

func trimNulls(b []byte) []byte {
	for len(b) > 0 && b[0] == 0 {
		b = b[1:]
	}
	return b
}

type ArrowStreams struct {
	Client        *Client
	OrderStream   *OrderStream
	DataStream    *DataStream
	HFTDataStream *HFTDataStream // set by NewStreamsWithHFT when used; optional (see hft_stream.go)
}

// NewStreamsOrderOnly connects only the order-updates WebSocket (wss://order-updates.arrow.trade).
func (c *Client) NewStreamsOrderOnly() (*ArrowStreams, error) {
	orderStream, err := c.ConnectOrderStream()
	if err != nil {
		return nil, err
	}
	return &ArrowStreams{
		Client:      c,
		OrderStream: orderStream,
		DataStream:  nil,
	}, nil
}

func (c *Client) NewStreams() (*ArrowStreams, error) {
	orderStream, err := c.ConnectOrderStream()
	if err != nil {
		return nil, err
	}
	dataStream, err := c.ConnectDataStream()
	if err != nil {
		_ = orderStream.Close()
		return nil, err
	}
	return &ArrowStreams{
		Client:      c,
		OrderStream: orderStream,
		DataStream:  dataStream,
	}, nil
}

// NewStreamsWithHFT connects order updates (wss://order-updates.arrow.trade) and the HFT market socket (wss://socket.arrow.trade).
// It does not open the standard token data stream (wss://ds.arrow.trade); use NewStreams for order + standard market data.
func (c *Client) NewStreamsWithHFT() (*ArrowStreams, error) {
	orderStream, err := c.ConnectOrderStream()
	if err != nil {
		return nil, err
	}
	hft, err := c.ConnectHFTDataStream()
	if err != nil {
		_ = orderStream.Close()
		return nil, err
	}
	return &ArrowStreams{
		Client:        c,
		OrderStream:   orderStream,
		DataStream:    nil,
		HFTDataStream: hft,
	}, nil
}

func (s *ArrowStreams) Close() error {
	var closeErr error
	if s.OrderStream != nil {
		if err := s.OrderStream.Close(); err != nil {
			closeErr = err
		}
	}
	if s.DataStream != nil {
		if err := s.DataStream.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	if s.HFTDataStream != nil {
		if err := s.HFTDataStream.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func beI32(data []byte) int32 {
	return int32(binary.BigEndian.Uint32(data))
}

func beI64(data []byte) int64 {
	return int64(binary.BigEndian.Uint64(data))
}

func StartKeepAlive(ctx context.Context, conn *websocket.Conn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = conn.WriteMessage(websocket.TextMessage, []byte("PONG"))
		}
	}
}
