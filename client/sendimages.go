package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

// ImageInput: một ảnh gửi theo URL công khai + kích thước THẬT (0 = Zalo bóp khung vuông 2560).
type ImageInput struct {
	URL    string
	Width  int
	Height int
}

// MaxAlbumImages: trần ảnh một album (Zalo PC cho chọn tối đa 50; giữ thấp để một lượt gửi không quá lâu).
const MaxAlbumImages = 20

// SendImages gửi nhiều ảnh thành MỘT album (như chọn nhiều ảnh trên Zalo PC) — mỗi ảnh vẫn là một tin Zalo riêng
// (có msgId riêng), cùng groupLayoutId nên khách thấy một khối lưới. Chú thích gắn ảnh đầu. Một ảnh → SendImage.
// Trả kết quả theo thứ tự ảnh; lỗi giữa chừng trả phần đã gửi kèm lỗi.
func (c *Client) SendImages(ctx context.Context, threadID string, threadType inbound.ThreadType, images []ImageInput, caption string) ([]SendResult, error) {
	if strings.TrimSpace(threadID) == "" {
		return nil, errors.New("zalo-kit: thread ID is required")
	}
	if len(images) == 0 {
		return nil, errors.New("zalo-kit: no image to send")
	}
	if len(images) > MaxAlbumImages {
		return nil, fmt.Errorf("zalo-kit: at most %d images per album", MaxAlbumImages)
	}
	for _, img := range images {
		if !strings.HasPrefix(img.URL, "http://") && !strings.HasPrefix(img.URL, "https://") {
			return nil, errors.New("zalo-kit: image URL must be publicly reachable over HTTP(S)")
		}
	}
	if len(images) == 1 {
		one, err := c.SendImage(ctx, threadID, threadType, images[0].URL, caption, images[0].Width, images[0].Height)
		if err != nil {
			return nil, err
		}
		return []SendResult{one}, nil
	}
	items := make([]zago.ImageItem, 0, len(images))
	for _, img := range images {
		items = append(items, zago.ImageItem{URL: img.URL, Width: img.Width, Height: img.Height})
	}
	var message *zago.Message
	if strings.TrimSpace(caption) != "" {
		message = &zago.Message{Text: caption}
	}
	c.mu.Lock()
	raws, err := c.api.SendMultiImageSized(items, threadID, zaloThreadType(threadType), message, 0)
	c.mu.Unlock()
	out := make([]SendResult, 0, len(raws))
	for _, raw := range raws {
		ids := extractIDs(raw)
		out = append(out, SendResult{MessageID: ids["msgId"], ClientMessageID: ids["cliMsgId"], Raw: raw})
	}
	if err != nil {
		return out, fmt.Errorf("send Zalo album (%d/%d sent): %w", len(out), len(images), err)
	}
	return out, nil
}
