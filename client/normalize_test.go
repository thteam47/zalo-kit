package client

import (
	"testing"
	"time"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

func TestNormalizeMessage(t *testing.T) {
	data := zago.NewMessageObject(map[string]any{
		"msgId": "m-1", "uidFrom": "u-1", "content": "xin chao",
		"msgType": "chat.photo", "href": "https://cdn/image.jpg", "ts": float64(1_700_000_000_000),
	})
	got := normalizeMessage("account-1", "self", "", "u-1", "", data, "group-1", zago.ThreadTypeGROUP, time.Time{})
	if got.ID != "m-1" || got.SenderID != "u-1" || got.ThreadType != inbound.ThreadGroup || got.Type != inbound.MessageImage {
		t.Fatalf("unexpected message: %#v", got)
	}
	if !got.Valid() {
		t.Fatalf("message should be valid: %#v", got)
	}
}

func TestExtractIDs(t *testing.T) {
	got := extractIDs(map[string]any{"globalMsgId": float64(123), "clientId": "456"})
	if got["msgId"] != "123" || got["cliMsgId"] != "456" {
		t.Fatalf("unexpected IDs: %#v", got)
	}
}

func TestResponseCode(t *testing.T) {
	if got := responseCode(map[string]any{"error_code": float64(0)}); got != 0 {
		t.Fatalf("got %d", got)
	}
}
