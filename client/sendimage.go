package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

// SendImage gửi một tấm ảnh kèm chú thích tuỳ chọn.
//
// Zalo KHÔNG nhận file tải lên ở đây: nó nhận một URL rồi tự đi lấy. Nghĩa là
// ảnh phải nằm ở địa chỉ công khai, Internet mở được — đường dẫn nội bộ hay
// link có chữ ký hết hạn đều làm khách thấy ảnh vỡ.
//
// width/height phải đúng với ảnh thật; sai thì cửa sổ chat hiện ảnh méo.
func (c *Client) SendImage(_ context.Context, threadID string, threadType inbound.ThreadType, imageURL, caption string, width, height int) (SendResult, error) {
	if strings.TrimSpace(threadID) == "" {
		return SendResult{}, errors.New("zalo-kit: thread ID is required")
	}
	if !strings.HasPrefix(imageURL, "http://") && !strings.HasPrefix(imageURL, "https://") {
		return SendResult{}, errors.New("zalo-kit: image URL must be publicly reachable over HTTP(S)")
	}
	var message *zago.Message
	if strings.TrimSpace(caption) != "" {
		message = &zago.Message{Text: caption}
	}
	c.mu.Lock()
	raw, err := c.api.SendImage(imageURL, threadID, zaloThreadType(threadType), width, height, message, 0)
	c.mu.Unlock()
	if err != nil {
		return SendResult{}, fmt.Errorf("send Zalo image: %w", err)
	}
	ids := extractIDs(raw)
	return SendResult{MessageID: ids["msgId"], ClientMessageID: ids["cliMsgId"], Raw: raw}, nil
}
