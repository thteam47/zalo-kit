package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/thteam47/zalo-kit/inbound"
)

// Hai loại gói tin đi chung đường với tin nhắn nhưng KHÔNG phải tin nhắn.

// reactionOf nhận ra sự kiện thả cảm xúc.
//
// za-go đẩy gói cảm xúc (cmd 612) qua CÙNG bộ xử lý với tin nhắn. Nội dung là
// {rMsg:[{gMsgID, cMsgID, msgType}], rIcon, rType, source} — không có chữ nào,
// nên trước đây nó thành một tin rỗng mới trong hội thoại.
func reactionOf(raw map[string]any) *inbound.Reaction {
	content := contentMap(raw["content"])
	if content == nil {
		return nil
	}
	_, hasIcon := content["rIcon"]
	targets, hasTargets := content["rMsg"].([]any)
	if !hasIcon && !hasTargets {
		return nil
	}
	reaction := &inbound.Reaction{Icon: strings.TrimSpace(fmt.Sprint(content["rIcon"]))}
	if reaction.Icon == "<nil>" {
		reaction.Icon = ""
	}
	if len(targets) > 0 {
		if target, ok := targets[0].(map[string]any); ok {
			reaction.TargetMessageID = firstID(target, "gMsgID", "msgId")
			reaction.TargetClientMessageID = firstID(target, "cMsgID", "cliMsgId")
		}
	}
	return reaction
}

// callOf nhận ra bản ghi cuộc gọi.
//
// Zalo gửi cuộc gọi như một tin "chat.recommended" có content là object:
// action dạng "recommened.calltime" / "recommened.misscall", params là chuỗi
// JSON mang thời lượng và loại gọi. Có bản Zalo lại để title
// "sendBubbleMessage" và mô tả "Cuộc gọi" — bắt cả hai dạng.
func callOf(raw map[string]any) *inbound.Call {
	content := contentMap(raw["content"])
	if content == nil {
		return nil
	}
	action := strings.ToLower(asText(content["action"]))
	title := asText(content["title"])
	description := asText(content["description"])
	isCall := strings.Contains(action, "call") || title == "sendBubbleMessage" ||
		(action != "recommened.link" && (startsWithCall(title) || startsWithCall(description)))
	if !isCall {
		return nil
	}
	params := contentMap(content["params"])
	call := &inbound.Call{Missed: strings.Contains(action, "miss")}
	if params != nil {
		call.DurationSec = intOf(firstPresent(params, "duration", "callDuration", "time"))
		switch intOf(firstPresent(params, "calltype", "callType", "type")) {
		case 1:
			call.Video = true
		}
		if intOf(firstPresent(params, "isVideo", "video")) == 1 {
			call.Video = true
		}
		if intOf(firstPresent(params, "missed", "isMissed")) == 1 {
			call.Missed = true
		}
	}
	text := strings.ToLower(title + " " + description)
	if strings.Contains(text, "nhỡ") || strings.Contains(text, "missed") {
		call.Missed = true
	}
	if strings.Contains(text, "video") {
		call.Video = true
	}
	return call
}

// CallSummary là dòng chữ người đọc thấy cho một cuộc gọi.
func CallSummary(call inbound.Call) string {
	kind := "Cuộc gọi thoại"
	if call.Video {
		kind = "Cuộc gọi video"
	}
	if call.Missed {
		return kind + " nhỡ"
	}
	if call.DurationSec > 0 {
		return kind + " · " + durationText(call.DurationSec)
	}
	return kind
}

func durationText(seconds int) string {
	if seconds < 60 {
		return strconv.Itoa(seconds) + " giây"
	}
	minutes, rest := seconds/60, seconds%60
	if minutes >= 60 {
		return fmt.Sprintf("%d giờ %d phút", minutes/60, minutes%60)
	}
	if rest == 0 {
		return strconv.Itoa(minutes) + " phút"
	}
	return fmt.Sprintf("%d phút %d giây", minutes, rest)
}

func startsWithCall(text string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(text)), "cuộc gọi")
}

// contentMap đọc một trường có thể là object hoặc chuỗi JSON của object.
func contentMap(value any) map[string]any {
	switch v := value.(type) {
	case map[string]any:
		return v
	case string:
		text := strings.TrimSpace(v)
		if !strings.HasPrefix(text, "{") {
			return nil
		}
		var out map[string]any
		decoder := json.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		if err := decoder.Decode(&out); err != nil {
			return nil
		}
		return out
	}
	return nil
}

func firstPresent(raw map[string]any, keys ...string) any {
	for _, key := range keys {
		if v, ok := raw[key]; ok && v != nil {
			return v
		}
	}
	return nil
}

// intOf đọc số kể cả json.Number (za-go giải mã bằng UseNumber) và chuỗi.
func intOf(value any) int {
	switch v := value.(type) {
	case nil:
		return 0
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	case bool:
		if v {
			return 1
		}
	}
	return 0
}
