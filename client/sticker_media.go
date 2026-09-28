package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/thteam47/zalo-kit/inbound"
)

// Tin nhãn dán của Zalo KHÔNG mang ảnh: content chỉ có {id, catId, type}. Không
// gắn ảnh thì tin lưu xuống không có MediaURL, và hộp thư chỉ in được chữ
// "[Nhãn dán]".
//
// ⚠️ Bản đầu đi hỏi sticker_detail rồi đoán khoá stickerWebpUrl/stickerUrl trong
// phản hồi — chạy thật thì KHÔNG ra ảnh (phản hồi qua ParseEnvelope không đúng
// hình dạng đã đoán), mà lỗi thì bị nuốt nên không ai biết. Zalo web hiện nhãn
// dán bằng một đường dẫn TĨNH chỉ cần mã nhãn (đo 29/09/2026: mã thật trả ảnh,
// mã không tồn tại trả 404). Dựng thẳng link đó: không gọi mạng trong vòng nhận
// tin, không phụ thuộc hình dạng phản hồi.
const stickerImageBase = "https://zalo-api.zadn.vn/api/emoticon/sticker/webpc"

// StickerImageURL là ảnh của một nhãn dán theo mã.
func StickerImageURL(stickerID int) string {
	if stickerID <= 0 {
		return ""
	}
	return fmt.Sprintf("%s?eid=%d&size=130", stickerImageBase, stickerID)
}

// stickerIDOf đọc mã nhãn dán trong content của tin (số hoặc chuỗi số).
func stickerIDOf(raw map[string]any) int {
	var content map[string]any
	switch value := raw["content"].(type) {
	case map[string]any:
		content = value
	case string:
		// Có đường đi đưa content xuống dạng chuỗi JSON thay vì object.
		if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &content); err != nil {
			return 0
		}
	default:
		return 0
	}
	// ⚠️ za-go giải JSON bằng UseNumber: số đến dưới dạng json.Number, KHÔNG
	// phải float64. Bản đầu chỉ bắt float64/int/string nên mọi nhãn dán thật đều
	// ra mã 0 → không có ảnh. cleanID đọc được mọi kiểu số.
	n, _ := strconv.Atoi(cleanID(content["id"]))
	return n
}

// fillStickerMedia gắn ảnh cho tin nhãn dán còn thiếu MediaURL.
func (c *Client) fillStickerMedia(msg inbound.Message, raw map[string]any) inbound.Message {
	if msg.Type != inbound.MessageSticker || msg.MediaURL != "" {
		return msg
	}
	msg.MediaURL = StickerImageURL(stickerIDOf(raw))
	return msg
}
