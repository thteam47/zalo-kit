package client

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

func normalizeRaw(t *testing.T, raw map[string]any) inbound.Message {
	t.Helper()
	return normalizeMessage("acc", "self", "", "", "", zago.NewMessageObject(raw), "thread", zago.ThreadTypeUSER, time.Now())
}

// ⚠️ Thả tim trên điện thoại từng hiện thành một bong bóng "[Tin nhắn]" rỗng:
// za-go đẩy gói cảm xúc qua cùng đường với tin nhắn.
func TestReactionIsNotAMessage(t *testing.T) {
	msg := normalizeRaw(t, map[string]any{
		"msgId": json.Number("7001"), "uidFrom": "self", "msgType": "chat.reaction", "ts": json.Number("1759140000000"),
		"content": map[string]any{
			"rIcon": "/-heart", "rType": json.Number("5"), "source": json.Number("6"),
			"rMsg": []any{map[string]any{"gMsgID": json.Number("6123"), "cMsgID": json.Number("99"), "msgType": json.Number("1")}},
		},
	})
	if msg.Type != inbound.MessageReaction || msg.Reaction == nil {
		t.Fatalf("type = %q reaction = %v, want reaction", msg.Type, msg.Reaction)
	}
	if msg.Reaction.Icon != "/-heart" || msg.Reaction.TargetMessageID != "6123" || msg.Reaction.TargetClientMessageID != "99" {
		t.Fatalf("reaction = %+v", *msg.Reaction)
	}
	if msg.Text != "" {
		t.Fatalf("reaction must carry no text, got %q", msg.Text)
	}
}

func TestReactionRemovedHasEmptyIcon(t *testing.T) {
	msg := normalizeRaw(t, map[string]any{
		"msgId": "7002", "uidFrom": "u1", "ts": "1759140000000",
		"content": `{"rIcon":"","rType":-1,"rMsg":[{"gMsgID":6123,"cMsgID":99}]}`,
	})
	if msg.Type != inbound.MessageReaction || msg.Reaction.Icon != "" || msg.Reaction.TargetMessageID != "6123" {
		t.Fatalf("got type=%q reaction=%+v", msg.Type, msg.Reaction)
	}
}

// ⚠️ Cuộc gọi từng hiện thành chữ "sendBubbleMessage — Cuộc gọi".
func TestCallBubbleBecomesReadable(t *testing.T) {
	msg := normalizeRaw(t, map[string]any{
		"msgId": "7003", "uidFrom": "u1", "ts": "1759140000000", "msgType": "chat.recommended",
		"content": map[string]any{"title": "sendBubbleMessage", "description": "Cuộc gọi"},
	})
	if msg.Type != inbound.MessageCall || msg.Text != "Cuộc gọi thoại" {
		t.Fatalf("got type=%q text=%q", msg.Type, msg.Text)
	}
}

func TestCallWithParams(t *testing.T) {
	msg := normalizeRaw(t, map[string]any{
		"msgId": "7004", "uidFrom": "u1", "ts": "1759140000000", "msgType": "chat.recommended",
		"content": map[string]any{"action": "recommened.calltime", "params": `{"duration":83,"calltype":1}`},
	})
	if msg.Type != inbound.MessageCall || msg.Text != "Cuộc gọi video · 1 phút 23 giây" {
		t.Fatalf("got type=%q text=%q", msg.Type, msg.Text)
	}
	missed := normalizeRaw(t, map[string]any{
		"msgId": "7005", "uidFrom": "u1", "ts": "1759140000000", "msgType": "chat.recommended",
		"content": map[string]any{"action": "recommened.misscall", "params": map[string]any{"duration": json.Number("0")}},
	})
	if missed.Text != "Cuộc gọi thoại nhỡ" {
		t.Fatalf("missed text = %q", missed.Text)
	}
}

func TestSharedLinkIsNotACall(t *testing.T) {
	msg := normalizeRaw(t, map[string]any{
		"msgId": "7006", "uidFrom": "u1", "ts": "1759140000000", "msgType": "chat.recommended",
		"content": map[string]any{"action": "recommened.link", "title": "Cuộc gọi video đẹp nhất năm", "href": "https://x"},
	})
	if msg.Type == inbound.MessageCall || msg.Type == inbound.MessageReaction {
		t.Fatalf("link misclassified as %q", msg.Type)
	}
}
