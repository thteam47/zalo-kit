package client

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

func normalizeMessage(accountID, selfID, mid, userID, text string, data *zago.MessageObject, threadID string, tt zago.ThreadType, fallback time.Time) inbound.Message {
	raw := map[string]any{}
	if data != nil {
		raw = data.ToMap()
	}
	msg := inbound.Message{
		ID:         firstID(raw, "msgId", "globalMsgId", "cliMsgId"),
		AccountID:  accountID,
		ThreadID:   cleanID(threadID),
		SenderID:   cleanID(userID),
		ThreadType: inbound.ThreadDirect,
		Type:       classifyMessage(raw),
		Text:       contentText(raw),
		MediaURL:   contentMedia(raw),
		OccurredAt: parseTime(raw, fallback),
	}
	if msg.ID == "" {
		msg.ID = cleanID(mid)
	}
	if msg.Text == "" && !looksLikeDump(text) {
		msg.Text = strings.TrimSpace(text)
	}
	if msg.SenderID == "" {
		msg.SenderID = firstID(raw, "uidFrom", "senderId", "userId")
	}
	if tt == zago.ThreadTypeGROUP {
		msg.ThreadType = inbound.ThreadGroup
	}
	msg.IsSelf = msg.SenderID != "" && cleanID(selfID) == msg.SenderID
	return msg
}

// contentText đọc phần chữ của tin.
//
// Với tin chia sẻ link hay ảnh, Zalo để content là một object chứ không phải
// chuỗi, và zago đưa xuống đây bằng fmt.Sprint nên ra nguyên khối
// "map[action: childnumber:0 description: href:https://... title:...]".
// Dán khối đó vào hội thoại thì người đọc không hiểu gì, còn bot thì tốn token
// đọc rác. Ở đây chỉ lấy tiêu đề và mô tả — đúng phần người ta thật sự viết.
func contentText(raw map[string]any) string {
	switch content := raw["content"].(type) {
	case string:
		if trimmed := strings.TrimSpace(content); trimmed != "" {
			return trimmed
		}
	case map[string]any:
		return joinParts(asText(content["title"]), asText(content["description"]))
	}
	return firstString(raw, "message", "text", "title")
}

// contentMedia lấy ảnh/link đính kèm, ưu tiên bên trong content vì tin chia sẻ
// để href ở đó chứ không phải ở gốc.
func contentMedia(raw map[string]any) string {
	if content, ok := raw["content"].(map[string]any); ok {
		if url := firstString(content, "href", "url", "thumb", "thumbnail"); url != "" {
			return url
		}
	}
	return firstString(raw, "href", "url", "thumb", "thumbnail")
}

// looksLikeDump nhận ra chuỗi do fmt.Sprint in ra từ map — thà để trống còn hơn
// lưu một khối rác vào lịch sử hội thoại.
func looksLikeDump(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "map[")
}

func asText(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func joinParts(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && !contains(kept, part) {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, " — ")
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func classifyMessage(raw map[string]any) inbound.MessageType {
	t := strings.ToLower(firstString(raw, "msgType", "type", "contentType"))
	switch {
	case strings.Contains(t, "photo"), strings.Contains(t, "image"):
		return inbound.MessageImage
	case strings.Contains(t, "file"), strings.Contains(t, "attach"):
		return inbound.MessageFile
	case strings.Contains(t, "sticker"):
		return inbound.MessageSticker
	case t == "", strings.Contains(t, "text"), strings.Contains(t, "chat"):
		return inbound.MessageText
	default:
		return inbound.MessageUnknown
	}
}

func parseTime(raw map[string]any, fallback time.Time) time.Time {
	v := firstID(raw, "ts", "timestamp", "time", "sendTime")
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	if n < 1_000_000_000_000 {
		n *= 1000
	}
	return time.UnixMilli(n).UTC()
}

func responseCode(raw map[string]any) int {
	for _, key := range []string{"error_code", "errorCode", "code"} {
		if v, ok := raw[key]; ok {
			n, _ := strconv.Atoi(cleanID(v))
			return n
		}
	}
	return -1
}

func extractIDs(raw any) map[string]string {
	out := map[string]string{}
	var data map[string]any
	switch v := raw.(type) {
	case map[string]any:
		data = v
	case interface{ ToMap() map[string]any }:
		data = v.ToMap()
	case interface{ ToDict() map[string]any }:
		data = v.ToDict()
	}
	if data == nil {
		return out
	}
	out["msgId"] = firstID(data, "msgId", "globalMsgId", "MsgId")
	out["cliMsgId"] = firstID(data, "cliMsgId", "clientId", "CliMsgId")
	return out
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := raw[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstID(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			if id := cleanID(v); id != "" {
				return id
			}
		}
	}
	return ""
}

func cleanID(v any) string {
	var s string
	switch n := v.(type) {
	case float64:
		s = strconv.FormatFloat(n, 'f', -1, 64)
	case float32:
		s = strconv.FormatFloat(float64(n), 'f', -1, 32)
	default:
		s = fmt.Sprint(v)
	}
	s = strings.TrimSpace(s)
	if s == "" || s == "<nil>" || s == "0" {
		return ""
	}
	return strings.TrimSuffix(s, ".0")
}
