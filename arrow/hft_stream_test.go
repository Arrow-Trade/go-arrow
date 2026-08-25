package arrow

import (
	"encoding/binary"
	"encoding/json"
	"testing"
)

func makeHFTCASFrame() []byte {
	buf := make([]byte, hftSizeCAS)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(hftSizeCAS))
	buf[2] = hftPktCAS
	buf[3] = HFTExchNSECM
	binary.LittleEndian.PutUint32(buf[4:8], 2885)
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint64(buf[8+i*8:16+i*8], uint64(100000+i))
		binary.LittleEndian.PutUint32(buf[40+i*4:44+i*4], uint32(10+i))
		binary.LittleEndian.PutUint64(buf[56+i*8:64+i*8], uint64(101000+i))
		binary.LittleEndian.PutUint32(buf[88+i*4:92+i*4], uint32(20+i))
	}
	binary.LittleEndian.PutUint64(buf[104:112], 1_700_000_000_000_000_000)
	var imb int64 = -5000
	binary.LittleEndian.PutUint64(buf[112:120], uint64(imb))
	binary.LittleEndian.PutUint64(buf[120:128], 100)
	binary.LittleEndian.PutUint64(buf[128:136], 97800)
	binary.LittleEndian.PutUint64(buf[136:144], 97750)
	binary.LittleEndian.PutUint64(buf[144:152], 0)
	binary.LittleEndian.PutUint32(buf[152:156], 1234)
	binary.LittleEndian.PutUint32(buf[156:160], 10)
	binary.LittleEndian.PutUint32(buf[160:164], 20)
	buf[164] = 2
	buf[165] = 1
	buf[166] = 2
	buf[167] = 1
	return buf
}

func TestHFTPacketMetaCAS(t *testing.T) {
	frame := makeHFTCASFrame()
	n, pkt, ok := hftPacketMeta(frame)
	if !ok || n != hftSizeCAS || pkt != hftPktCAS {
		t.Fatalf("meta n=%d pkt=%d ok=%v", n, pkt, ok)
	}
}

func TestParseHFTCAS(t *testing.T) {
	tick, err := parseHFTCAS(makeHFTCASFrame())
	if err != nil {
		t.Fatal(err)
	}
	if tick.PktType != hftPktCAS || tick.Token != 2885 {
		t.Fatalf("header: %+v", tick)
	}
	if tick.BidPx[0] != 100000 || tick.BidSize[3] != 13 {
		t.Fatalf("bids: px=%v size=%v", tick.BidPx, tick.BidSize)
	}
	if tick.AskPx[0] != 101000 || tick.AskSize[1] != 21 {
		t.Fatalf("asks: px=%v size=%v", tick.AskPx, tick.AskSize)
	}
	if tick.ImbalanceQty != -5000 {
		t.Fatalf("ImbalanceQty=%d", tick.ImbalanceQty)
	}
	if tick.IndicativePx != 97800 || tick.ClosingRefPx != 97750 {
		t.Fatalf("px indicative=%d ref=%d", tick.IndicativePx, tick.ClosingRefPx)
	}
	if tick.IndicativeQty != 1234 || !tick.OnlyLimitOrders {
		t.Fatalf("qty=%d onlyLimit=%v", tick.IndicativeQty, tick.OnlyLimitOrders)
	}
	if tick.Phase != 2 || tick.ImbalanceSide != 1 {
		t.Fatalf("phase=%d side=%d", tick.Phase, tick.ImbalanceSide)
	}
}

func TestDispatchHFTCAS(t *testing.T) {
	var got *HFTCASTick
	s := &HFTDataStream{}
	s.dispatchHFTPayload(makeHFTCASFrame(), nil, nil, func(t HFTCASTick) { got = &t }, nil, func(err error) {
		t.Fatal(err)
	})
	if got == nil || got.Token != 2885 || got.ImbalanceQty != -5000 {
		t.Fatalf("got=%+v", got)
	}
}

func TestNormalizeHFTModeCAS(t *testing.T) {
	m, err := normalizeHFTMode("cas")
	if err != nil || m != "cas" {
		t.Fatalf("mode=%q err=%v", m, err)
	}
	if _, err := normalizeHFTMode("nope"); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestHFTSubscribeMessageOmitsLatencyForCAS(t *testing.T) {
	cas := hftSubscribeMessage("cas", 200)
	if _, ok := cas["latency"]; ok {
		t.Fatalf("cas message should omit latency: %v", cas)
	}
	raw, err := json.Marshal(cas)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"code":"sub","mode":"cas"}` {
		t.Fatalf("cas json=%s", raw)
	}

	ltpc := hftSubscribeMessage("ltpc", 200)
	if ltpc["latency"] != 200 {
		t.Fatalf("ltpc latency=%v", ltpc["latency"])
	}
}

func TestParseHFTResponseModeCAS(t *testing.T) {
	buf := make([]byte, hftSizeResponse)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(hftSizeResponse))
	buf[4] = hftPktResponse
	copy(buf[6:22], []byte("SUCCESS"))
	buf[534] = 0
	buf[535] = 3
	binary.LittleEndian.PutUint16(buf[536:538], 1)
	r := parseHFTResponse(buf)
	if r.ModeStr != "cas" || r.RequestTypeStr != "subscribe" {
		t.Fatalf("resp=%+v", r)
	}
}
