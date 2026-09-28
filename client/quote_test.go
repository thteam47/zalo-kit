package client

import (
	"testing"
	"time"

	"github.com/thteam47/zalo-kit/inbound"
)

func TestQuoteOfDocKhoiTrichDan(t *testing.T) {
	raw := map[string]any{"quote": map[string]any{
		"ownerId": float64(123456789), "globalMsgId": float64(7100000000001),
		"cliMsgId": float64(1727500000000), "msg": " giá bao nhiêu ", "ts": float64(1727500000000),
	}}
	q := quoteOf(raw)
	if q == nil || q.OwnerID != "123456789" || q.MessageID != "7100000000001" || q.ClientMessageID != "1727500000000" || q.Text != "giá bao nhiêu" {
		t.Fatalf("đọc sai khối trích dẫn: %#v", q)
	}
	if !q.OccurredAt.Equal(time.UnixMilli(1727500000000).UTC()) {
		t.Fatalf("sai mốc giờ: %v", q.OccurredAt)
	}
	if quoteOf(map[string]any{"quote": "khong phai map"}) != nil || quoteOf(map[string]any{}) != nil {
		t.Fatal("không có khối trích dẫn thì phải nil")
	}
}

func TestQuoteInfoMacDinhWebchat(t *testing.T) {
	info := quoteInfo(inbound.Quote{MessageID: "1", ClientMessageID: "2", Text: "a",
		OccurredAt: time.UnixMilli(1727500000000)}, "owner")
	if info.MsgType != "webchat" || info.Ts != "1727500000000" || info.OwnerID != "owner" || info.CliMsgID != "2" {
		t.Fatalf("%#v", info)
	}
}
