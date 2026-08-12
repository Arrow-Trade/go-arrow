package arrow

import (
	"encoding/binary"
	"testing"
)

func buildFullPacket(depthOffset int, cas bool) []byte {
	size := depthOffset + 10*14
	if cas {
		size = 265
	}
	buf := make([]byte, size)
	binary.BigEndian.PutUint32(buf[0:4], 3045)
	binary.BigEndian.PutUint32(buf[4:8], 97780)
	binary.BigEndian.PutUint32(buf[13:17], 10)
	binary.BigEndian.PutUint32(buf[17:21], 14950)
	binary.BigEndian.PutUint64(buf[21:29], 1000)
	binary.BigEndian.PutUint64(buf[29:37], 2000)
	binary.BigEndian.PutUint32(buf[37:41], 14800)
	binary.BigEndian.PutUint32(buf[41:45], 15200)
	binary.BigEndian.PutUint32(buf[45:49], 14700)
	binary.BigEndian.PutUint32(buf[49:53], 14600)
	binary.BigEndian.PutUint64(buf[53:61], 50000)
	binary.BigEndian.PutUint32(buf[61:65], 123456)
	binary.BigEndian.PutUint32(buf[65:69], 123457)
	binary.BigEndian.PutUint64(buf[69:77], 100000)
	binary.BigEndian.PutUint64(buf[77:85], 110000)
	binary.BigEndian.PutUint64(buf[85:93], 90000)
	binary.BigEndian.PutUint32(buf[93:97], 87345)
	binary.BigEndian.PutUint32(buf[97:101], 106745)
	for i := 0; i < 10; i++ {
		off := depthOffset + i*14
		binary.BigEndian.PutUint64(buf[off:off+8], uint64(100+i))
		binary.BigEndian.PutUint32(buf[off+8:off+12], uint32(97750+i))
		binary.BigEndian.PutUint16(buf[off+12:off+14], 5)
	}
	if cas {
		var imb int64 = -12345
		binary.BigEndian.PutUint64(buf[249:257], uint64(imb))
		binary.BigEndian.PutUint32(buf[257:261], 97800)
		binary.BigEndian.PutUint32(buf[261:265], 97750)
	}
	return buf
}

func TestParseMarketTickFull249(t *testing.T) {
	payload := buildFullPacket(109, false)
	if len(payload) != 249 {
		t.Fatalf("len=%d, want 249", len(payload))
	}
	tick, err := ParseMarketTick(payload)
	if err != nil {
		t.Fatal(err)
	}
	if tick.Mode != StreamModeFull {
		t.Fatalf("mode=%q, want full", tick.Mode)
	}
	if len(tick.Bids) != 5 || tick.Bids[0].Price != 97750 {
		t.Fatalf("bids=%v", tick.Bids)
	}
	if tick.ImbalanceQty != 0 || tick.IndicativeClose != 0 || tick.RefPrice != 0 {
		t.Fatalf("unexpected CAS fields: %+v", tick)
	}
}

func TestParseMarketTickFull265CAS(t *testing.T) {
	payload := buildFullPacket(109, true)
	if len(payload) != 265 {
		t.Fatalf("len=%d, want 265", len(payload))
	}
	tick, err := ParseMarketTick(payload)
	if err != nil {
		t.Fatal(err)
	}
	if tick.Mode != StreamModeFull {
		t.Fatalf("mode=%q, want full", tick.Mode)
	}
	if tick.ImbalanceQty != -12345 {
		t.Fatalf("ImbalanceQty=%d, want -12345", tick.ImbalanceQty)
	}
	if tick.IndicativeClose != 97800 {
		t.Fatalf("IndicativeClose=%d, want 97800", tick.IndicativeClose)
	}
	if tick.RefPrice != 97750 {
		t.Fatalf("RefPrice=%d, want 97750", tick.RefPrice)
	}
	if len(tick.Bids) != 5 || tick.Bids[0].Price != 97750 {
		t.Fatalf("bids=%v", tick.Bids)
	}
}

func appendCAS(base []byte) []byte {
	out := make([]byte, len(base)+16)
	copy(out, base)
	off := len(base)
	var imb int64 = -12345
	binary.BigEndian.PutUint64(out[off:off+8], uint64(imb))
	binary.BigEndian.PutUint32(out[off+8:off+12], 97800)
	binary.BigEndian.PutUint32(out[off+12:off+16], 97750)
	return out
}

func TestParseMarketTickLTPCAS(t *testing.T) {
	base := make([]byte, 13)
	binary.BigEndian.PutUint32(base[0:4], 3045)
	binary.BigEndian.PutUint32(base[4:8], 97780)
	payload := appendCAS(base)
	if len(payload) != 29 {
		t.Fatalf("len=%d, want 29", len(payload))
	}
	tick, err := ParseMarketTick(payload)
	if err != nil {
		t.Fatal(err)
	}
	if tick.Mode != StreamModeLTP {
		t.Fatalf("mode=%q, want ltp", tick.Mode)
	}
	if tick.Close != 0 {
		t.Fatalf("close=%d, want 0 (CAS must not be read as close)", tick.Close)
	}
	if tick.ImbalanceQty != -12345 || tick.IndicativeClose != 97800 || tick.RefPrice != 97750 {
		t.Fatalf("CAS fields: %+v", tick)
	}
}

func TestParseMarketTickLTPCAS33(t *testing.T) {
	base := make([]byte, 17)
	binary.BigEndian.PutUint32(base[0:4], 3045)
	binary.BigEndian.PutUint32(base[4:8], 97780)
	binary.BigEndian.PutUint32(base[13:17], 97000)
	payload := appendCAS(base)
	tick, err := ParseMarketTick(payload)
	if err != nil {
		t.Fatal(err)
	}
	if tick.Mode != StreamModeLTPC || tick.Close != 97000 {
		t.Fatalf("tick=%+v", tick)
	}
	if tick.ImbalanceQty != -12345 {
		t.Fatalf("ImbalanceQty=%d", tick.ImbalanceQty)
	}
}

func TestParseMarketTickQuoteCAS109(t *testing.T) {
	base := make([]byte, 93)
	binary.BigEndian.PutUint32(base[0:4], 3045)
	binary.BigEndian.PutUint32(base[4:8], 97780)
	binary.BigEndian.PutUint32(base[13:17], 10)
	binary.BigEndian.PutUint32(base[45:49], 97000)
	payload := appendCAS(base)
	if len(payload) != 109 {
		t.Fatalf("len=%d, want 109", len(payload))
	}
	tick, err := ParseMarketTick(payload)
	if err != nil {
		t.Fatal(err)
	}
	if tick.Mode != StreamModeQuote || tick.Close != 97000 {
		t.Fatalf("tick=%+v", tick)
	}
	if tick.ImbalanceQty != -12345 || tick.IndicativeClose != 97800 || tick.RefPrice != 97750 {
		t.Fatalf("CAS fields: %+v", tick)
	}
	if len(tick.Bids) != 0 {
		t.Fatalf("quote should have no depth, got %v", tick.Bids)
	}
}
