package client

import (
	"strconv"
	"strings"
	"sync"

	"github.com/thteam47/zalo-kit/inbound"
)

// Tin nhãn dán của Zalo KHÔNG mang ảnh: content chỉ có {id, catId, type}. Không
// đi hỏi thì tin lưu xuống không có MediaURL, và hộp thư chỉ in được chữ
// "[Nhãn dán]". Ảnh thật nằm ở sticker_detail (stickerWebpUrl / stickerUrl).
//
// Bộ đệm theo MÃ nhãn dán, dùng chung mọi tài khoản: một mã nhãn luôn trỏ cùng
// một ảnh, và khách hay gửi đi gửi lại vài nhãn quen — hỏi lại mỗi lần là tự
// chuốc giới hạn của Zalo.
var stickerURLCache sync.Map // int -> string

// stickerIDOf đọc mã nhãn dán trong content của tin (số hoặc chuỗi số).
func stickerIDOf(raw map[string]any) int {
	content, ok := raw["content"].(map[string]any)
	if !ok {
		return 0
	}
	switch id := content["id"].(type) {
	case float64:
		return int(id)
	case int:
		return id
	case int64:
		return int(id)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(id))
		return n
	}
	return 0
}

// findStickerURL duyệt phản hồi sticker_detail lấy đường dẫn ảnh. Zalo lồng
// dữ liệu nhiều lớp tuỳ phiên bản, nên duyệt cả cây thay vì đoán một dạng.
// Ưu tiên webp (có chuyển động, trình duyệt nào cũng xem được).
func findStickerURL(value any, depth int) string {
	if depth > 6 {
		return ""
	}
	switch v := value.(type) {
	case map[string]any:
		for _, key := range []string{"stickerWebpUrl", "stickerUrl"} {
			if s, ok := v[key].(string); ok && strings.HasPrefix(strings.TrimSpace(s), "http") {
				return strings.TrimSpace(s)
			}
		}
		for _, child := range v {
			if s := findStickerURL(child, depth+1); s != "" {
				return s
			}
		}
	case []any:
		for _, child := range v {
			if s := findStickerURL(child, depth+1); s != "" {
				return s
			}
		}
	}
	return ""
}

// stickerURL trả ảnh của mã nhãn dán, hỏi Zalo MỘT lần cho mỗi mã. Hỏng thì trả
// rỗng — thiếu ảnh không được phép làm mất tin.
func (c *Client) stickerURL(id int) string {
	if id <= 0 {
		return ""
	}
	if cached, ok := stickerURLCache.Load(id); ok {
		return cached.(string)
	}
	detail, err := c.StickerDetail(id)
	if err != nil {
		return ""
	}
	url := findStickerURL(detail, 0)
	if url != "" {
		stickerURLCache.Store(id, url)
	}
	return url
}

// fillStickerMedia gắn ảnh cho tin nhãn dán còn thiếu MediaURL.
func (c *Client) fillStickerMedia(msg inbound.Message, raw map[string]any) inbound.Message {
	if msg.Type != inbound.MessageSticker || msg.MediaURL != "" {
		return msg
	}
	msg.MediaURL = c.stickerURL(stickerIDOf(raw))
	return msg
}
