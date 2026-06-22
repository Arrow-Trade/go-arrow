package arrow

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

const hftStreamURL = "wss://socket.arrow.trade"

// HFT exchange segments (symIds / wire exch_seg). See Arrow WebSocket docs.
const (
	HFTExchNSECM = 0 // NSE cash
	HFTExchNSEFO = 1 // NSE F&O
	HFTExchBSECM = 2 // BSE cash
	HFTExchBSEFO = 3 // BSE F&O
)

const (
	hftPktLTP      = 1
	hftPktFull     = 2
	hftPktResponse = 99

	hftSizeLTP      = 40
	hftSizeFull     = 196
	hftSizeResponse = 540
)

// HFTDataStream is the low-latency market WebSocket (wss://socket.arrow.trade).
// Wire format matches the Python client: little-endian binary inbound, JSON sub/unsub outbound.
// See https://docs.arrow.trade/python-sdk/websocket-streaming/ (HFT Data Stream).
type HFTDataStream struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *Client) ConnectHFTDataStream() (*HFTDataStream, error) {
	q := url.Values{}
	q.Set("appID", c.Config.AppID)
	q.Set("token", c.Config.Token)
	q.Set("zstd", "1")
	u := fmt.Sprintf("%s?%s", hftStreamURL, q.Encode())
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return nil, err
	}
	return &HFTDataStream{conn: conn}, nil
}

func (s *HFTDataStream) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

func normalizeHFTMode(mode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "l", "ltpc":
		return "ltpc", nil
	case "f", "full":
		return "full", nil
	default:
		return "", fmt.Errorf("hft mode must be ltpc, l, full, or f, got %q", mode)
	}
}

// SubscribeHFTSymbols subscribes by symbol strings (e.g. NSE.SBIN-EQ). latencyMs is tick spacing (50–60000).
func (s *HFTDataStream) SubscribeHFTSymbols(mode string, symbols []string, latencyMs int) error {
	m, err := normalizeHFTMode(mode)
	if err != nil {
		return err
	}
	if len(symbols) == 0 {
		return errors.New("hft subscribe: empty symbols")
	}
	msg := map[string]any{
		"code":    "sub",
		"mode":    m,
		"latency": latencyMs,
		"symbols": symbols,
	}
	return s.writeJSON(msg)
}

// SubscribeHFTTokens subscribes integer instrument IDs on a single exchange segment (default NSE cash = 0).
func (s *HFTDataStream) SubscribeHFTTokens(mode string, exchSeg int, ids []int32, latencyMs int) error {
	m, err := normalizeHFTMode(mode)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return errors.New("hft subscribe: empty ids")
	}
	// JSON numbers are float64 in generic decode; server accepts int array.
	arr := make([]int32, len(ids))
	copy(arr, ids)
	msg := map[string]any{
		"code":    "sub",
		"mode":    m,
		"latency": latencyMs,
		"symIds": []map[string]any{
			{"exch_seg": exchSeg, "ids": arr},
		},
	}
	return s.writeJSON(msg)
}

// SubscribeHFTBySegment subscribes token IDs grouped by exchange segment.
func (s *HFTDataStream) SubscribeHFTBySegment(mode string, segments map[int][]int32, latencyMs int) error {
	m, err := normalizeHFTMode(mode)
	if err != nil {
		return err
	}
	var symIDs []map[string]any
	for seg, ids := range segments {
		if len(ids) == 0 {
			continue
		}
		cp := make([]int32, len(ids))
		copy(cp, ids)
		symIDs = append(symIDs, map[string]any{"exch_seg": seg, "ids": cp})
	}
	if len(symIDs) == 0 {
		return errors.New("hft subscribe: no ids in segments")
	}
	msg := map[string]any{
		"code":    "sub",
		"mode":    m,
		"latency": latencyMs,
		"symIds":  symIDs,
	}
	return s.writeJSON(msg)
}

// UnsubscribeHFTSymbols sends unsub for symbol strings.
func (s *HFTDataStream) UnsubscribeHFTSymbols(mode string, symbols []string) error {
	m, err := normalizeHFTMode(mode)
	if err != nil {
		return err
	}
	msg := map[string]any{"code": "unsub", "mode": m, "symbols": symbols}
	return s.writeJSON(msg)
}

// UnsubscribeHFTTokens sends unsub for token IDs on one segment.
func (s *HFTDataStream) UnsubscribeHFTTokens(mode string, exchSeg int, ids []int32) error {
	m, err := normalizeHFTMode(mode)
	if err != nil {
		return err
	}
	arr := make([]int32, len(ids))
	copy(arr, ids)
	msg := map[string]any{
		"code": "unsub",
		"mode": m,
		"symIds": []map[string]any{
			{"exch_seg": exchSeg, "ids": arr},
		},
	}
	return s.writeJSON(msg)
}

func (s *HFTDataStream) writeJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteMessage(websocket.TextMessage, b)
}

// ReadHFT dispatches binary HFT packets until ctx is done or the socket errors.
// Text frames (e.g. keepalives) are ignored. Callbacks may be nil.
func (s *HFTDataStream) ReadHFT(ctx context.Context, onLTP func(HFTLTPTick), onFull func(HFTFullTick), onResponse func(HFTResponsePacket), onError func(error)) {
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
		if mt != websocket.BinaryMessage {
			continue
		}
		if len(payload) < 3 {
			continue
		}
		frameSize := int(int16(binary.LittleEndian.Uint16(payload[0:2])))
		if frameSize > 0 && len(payload) < frameSize {
			if onError != nil {
				onError(fmt.Errorf("hft incomplete frame: want %d bytes, got %d", frameSize, len(payload)))
			}
			continue
		}
		pkt := payload[2]
		switch pkt {
		case hftPktResponse:
			if len(payload) >= hftSizeResponse {
				if onResponse != nil {
					onResponse(parseHFTResponse(payload))
				}
			}
		case hftPktLTP:
			if len(payload) >= hftSizeLTP {
				if onLTP != nil {
					t, perr := parseHFTLTP(payload)
					if perr != nil {
						if onError != nil {
							onError(perr)
						}
						continue
					}
					onLTP(t)
				}
			}
		case hftPktFull:
			if len(payload) >= hftSizeFull {
				if onFull != nil {
					t, perr := parseHFTFull(payload)
					if perr != nil {
						if onError != nil {
							onError(perr)
						}
						continue
					}
					onFull(t)
				}
			}
		}
	}
}

// HFTLTPTick is one LTP-style packet (prices in paise).
type HFTLTPTick struct {
	Size    int16
	PktType uint8
	ExchSeg uint8
	Token   int32
	LTP     int32
	VWAP    int32
	Volume  int64
	LTT     uint64
	ATV     uint32
	BTV     uint32
}

func parseHFTLTP(data []byte) (HFTLTPTick, error) {
	if len(data) < hftSizeLTP {
		return HFTLTPTick{}, fmt.Errorf("hft ltp: need %d bytes", hftSizeLTP)
	}
	return HFTLTPTick{
		Size:    int16(binary.LittleEndian.Uint16(data[0:2])),
		PktType: data[2],
		ExchSeg: data[3],
		Token:   int32(binary.LittleEndian.Uint32(data[4:8])),
		LTP:     int32(binary.LittleEndian.Uint32(data[8:12])),
		VWAP:    int32(binary.LittleEndian.Uint32(data[12:16])),
		Volume:  int64(binary.LittleEndian.Uint64(data[16:24])),
		LTT:     binary.LittleEndian.Uint64(data[24:32]),
		ATV:     binary.LittleEndian.Uint32(data[32:36]),
		BTV:     binary.LittleEndian.Uint32(data[36:40]),
	}, nil
}

// HFTFullTick is one full-depth packet (prices in paise).
type HFTFullTick struct {
	Size    int16
	PktType uint8
	ExchSeg uint8
	Token   int32
	LTP     int32
	LTQ     int32
	VWAP    int32
	Open    int32
	High    int32
	Close   int32
	Low     int32
	LTT     int32
	DprL    int32
	DprH    int32
	TBQ     int64
	TSQ     int64
	Volume  int64
	BidPx   [5]int32
	AskPx   [5]int32
	BidSize [5]int32
	AskSize [5]int32
	BidOrd  [5]uint16
	AskOrd  [5]uint16
	OI      uint64
	TS      uint64
	ATV     uint32
	BTV     uint32
}

func parseHFTFull(data []byte) (HFTFullTick, error) {
	if len(data) < hftSizeFull {
		return HFTFullTick{}, fmt.Errorf("hft full: need %d bytes", hftSizeFull)
	}
	var t HFTFullTick
	t.Size = int16(binary.LittleEndian.Uint16(data[0:2]))
	t.PktType = data[2]
	t.ExchSeg = data[3]
	t.Token = int32(binary.LittleEndian.Uint32(data[4:8]))
	t.LTP = int32(binary.LittleEndian.Uint32(data[8:12]))
	t.LTQ = int32(binary.LittleEndian.Uint32(data[12:16]))
	t.VWAP = int32(binary.LittleEndian.Uint32(data[16:20]))
	t.Open = int32(binary.LittleEndian.Uint32(data[20:24]))
	t.High = int32(binary.LittleEndian.Uint32(data[24:28]))
	t.Close = int32(binary.LittleEndian.Uint32(data[28:32]))
	t.Low = int32(binary.LittleEndian.Uint32(data[32:36]))
	t.LTT = int32(binary.LittleEndian.Uint32(data[36:40]))
	t.DprL = int32(binary.LittleEndian.Uint32(data[40:44]))
	t.DprH = int32(binary.LittleEndian.Uint32(data[44:48]))
	t.TBQ = int64(binary.LittleEndian.Uint64(data[48:56]))
	t.TSQ = int64(binary.LittleEndian.Uint64(data[56:64]))
	t.Volume = int64(binary.LittleEndian.Uint64(data[64:72]))
	copy(t.BidPx[:], decodeI32Slice5(data[72:92]))
	copy(t.AskPx[:], decodeI32Slice5(data[92:112]))
	copy(t.BidSize[:], decodeI32Slice5(data[112:132]))
	copy(t.AskSize[:], decodeI32Slice5(data[132:152]))
	copy(t.BidOrd[:], decodeU16Slice5(data[152:162]))
	copy(t.AskOrd[:], decodeU16Slice5(data[162:172]))
	t.OI = binary.LittleEndian.Uint64(data[172:180])
	t.TS = binary.LittleEndian.Uint64(data[180:188])
	t.ATV = binary.LittleEndian.Uint32(data[188:192])
	t.BTV = binary.LittleEndian.Uint32(data[192:196])
	return t, nil
}

func decodeI32Slice5(b []byte) []int32 {
	out := make([]int32, 5)
	for i := range 5 {
		out[i] = int32(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func decodeU16Slice5(b []byte) []uint16 {
	out := make([]uint16, 5)
	for i := range 5 {
		out[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return out
}

// HFTResponsePacket is a subscribe/unsubscribe acknowledgement (binary pkt_type 99) from wss://socket.arrow.trade.
// ErrorCode / ErrorMsg describe subscription outcome; common values include SUCCESS, E_PARTIAL, E_ALL_INVALID,
// E_INVALID_JSON, E_MISSING_FIELD, E_INVALID_PARAM, E_PARSE_ERROR (see Arrow WebSocket HFT documentation).
// These strings are unrelated to REST historical errors (e.g. BadRequestError / invalid token on GET /candle/...).
type HFTResponsePacket struct {
	FrameSize      int16 // LE int16 at wire bytes 0–1 (expected 540 for a full response frame)
	PktType        uint8 // wire byte 2 (99)
	ExchSeg        uint8 // wire byte 3
	ErrorCode      string
	ErrorMsg       string
	RequestType    uint8 // 0=sub, 1=unsub (wire byte 534)
	Mode           uint8 // 0=ltpc, 1=full (wire byte 535)
	SuccessCount   uint16
	ErrorCount     uint16
	RequestTypeStr string
	ModeStr        string
}

func parseHFTResponse(data []byte) HFTResponsePacket {
	// String and tail field offsets match pyarrow_client HFTDataStream._parse_response (data[6:22], etc.).
	// Packet type for routing is wire byte 2 (see ReadHFT); bytes 4–5 are unused on the wire we model here.
	r := HFTResponsePacket{
		FrameSize:    int16(binary.LittleEndian.Uint16(data[0:2])),
		PktType:      data[2],
		ExchSeg:      data[3],
		ErrorCode:    strings.TrimRight(string(data[6:22]), "\x00"),
		ErrorMsg:     strings.TrimRight(string(data[22:534]), "\x00"),
		RequestType:  data[534],
		Mode:         data[535],
		SuccessCount: binary.LittleEndian.Uint16(data[536:538]),
		ErrorCount:   binary.LittleEndian.Uint16(data[538:540]),
	}
	switch r.RequestType {
	case 0:
		r.RequestTypeStr = "subscribe"
	case 1:
		r.RequestTypeStr = "unsubscribe"
	default:
		r.RequestTypeStr = "unknown"
	}
	switch r.Mode {
	case 0:
		r.ModeStr = "ltpc"
	case 1:
		r.ModeStr = "full"
	default:
		r.ModeStr = "unknown"
	}
	return r
}
