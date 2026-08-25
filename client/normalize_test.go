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

// Sự cố thật: hộp thư hiện nguyên khối "map[action: childnumber:0 ...]" cho tin
// chia sẻ link Shopee. Zalo để content là object, zago in ra bằng fmt.Sprint.
func TestSharedLinkKeepsOnlyReadableText(t *testing.T) {
	data := zago.NewMessageObject(map[string]any{
		"msgId": "m-2", "uidFrom": "u-9", "msgType": "chat.recommended",
		"ts": float64(1_700_000_000_000),
		"content": map[string]any{
			"title":       "Cọ chải cán dài lỗi giá 17k",
			"description": "Cọ 2 mặt, vừa tắm vừa cọ",
			"href":        "https://s.shopee.vn/5LBNKyg0S9",
			"thumb":       "https://photo-stal-7.zdn.vn/anh.jpg",
			"params":      `{"height":1280}`,
		},
	})
	got := normalizeMessage("account-1", "self", "", "u-9", "map[action: childnumber:0 title:...]", data, "u-9", zago.ThreadTypeUSER, time.Time{})
	if got.Text != "Cọ chải cán dài lỗi giá 17k — Cọ 2 mặt, vừa tắm vừa cọ" {
		t.Fatalf("phải lấy tiêu đề và mô tả, nhận %q", got.Text)
	}
	if got.MediaURL != "https://s.shopee.vn/5LBNKyg0S9" {
		t.Fatalf("phải lấy link trong content, nhận %q", got.MediaURL)
	}
}

// Không đọc được gì thì để trống, tuyệt đối không dán khối map vào hội thoại.
func TestDumpTextIsNeverStored(t *testing.T) {
	data := zago.NewMessageObject(map[string]any{
		"msgId": "m-3", "uidFrom": "u-9", "msgType": "chat.photo", "ts": float64(1_700_000_000_000),
		"content": map[string]any{"href": "https://cdn/anh.jpg"},
	})
	got := normalizeMessage("account-1", "self", "", "u-9", "map[action: href:https://cdn/anh.jpg]", data, "u-9", zago.ThreadTypeUSER, time.Time{})
	if got.Text != "" {
		t.Fatalf("không có chữ thì phải để trống, nhận %q", got.Text)
	}
	if got.Type != inbound.MessageImage {
		t.Fatalf("vẫn phải nhận ra là ảnh, nhận %q", got.Type)
	}
}

// Tin chữ bình thường không được đụng vào.
func TestPlainTextUnchanged(t *testing.T) {
	data := zago.NewMessageObject(map[string]any{
		"msgId": "m-4", "uidFrom": "u-9", "content": "còn hàng không shop", "ts": float64(1_700_000_000_000),
	})
	got := normalizeMessage("account-1", "self", "", "u-9", "", data, "u-9", zago.ThreadTypeUSER, time.Time{})
	if got.Text != "còn hàng không shop" {
		t.Fatalf("tin chữ phải giữ nguyên, nhận %q", got.Text)
	}
}
