package client

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

// Lớp bọc cho các loại tin ngoài chữ và ảnh.
//
// Vì sao bọc thay vì để bên gọi dùng thẳng zago: mọi lời gọi ở đây đều phải
// (1) khoá mu, (2) kiểm phiên còn sống, (3) đổi ThreadType của ta sang của
// zago. Ba bước đó lặp lại ở từng hàm, và quên bước 2 nghĩa là nil panic
// trong lúc đang chạy thay vì một lỗi đọc được.

// guard gom đúng ba bước nói trên. Trả về api đã kiểm hoặc lỗi.
func (c *Client) guard() (*zago.ZaloAPI, error) {
	if c.api == nil {
		return nil, ErrSessionInvalid
	}
	return c.api, nil
}

// SendFile gửi một tệp đã nằm trên kho công khai.
//
// ⚠️ fileURL phải là đường Zalo TỰ TẢI VỀ ĐƯỢC. Zalo không nhận byte tải lên ở
// đường này — nó đi lấy theo URL, nên một đường ký tạm hết hạn sau vài phút sẽ
// hỏng ngẫu nhiên tuỳ lúc Zalo đi lấy.
func (c *Client) SendFile(threadID string, threadType inbound.ThreadType,
	fileURL, fileName, extension string, fileSize int) error {
	if strings.TrimSpace(fileURL) == "" || strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thiếu tệp hoặc mã luồng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendFile(fileURL, threadID, zaloThreadType(threadType),
		fileName, fileSize, extension, "", "", 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi tệp: %w", err)
	}
	return nil
}

// SendVideo gửi video kèm ảnh nền.
func (c *Client) SendVideo(threadID string, threadType inbound.ThreadType,
	videoURL, thumbnailURL, caption string, durationSeconds, width, height int) error {
	if strings.TrimSpace(videoURL) == "" || strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thiếu video hoặc mã luồng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	var message *zago.Message
	if strings.TrimSpace(caption) != "" {
		built := zago.Message{Text: caption}
		message = &built
	}
	if _, err = api.SendVideo(videoURL, thumbnailURL, durationSeconds, threadID,
		zaloThreadType(threadType), width, height, message, 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi video: %w", err)
	}
	return nil
}

// SendVoice gửi tin thoại.
func (c *Client) SendVoice(threadID string, threadType inbound.ThreadType,
	voiceURL string, fileSize int) error {
	if strings.TrimSpace(voiceURL) == "" || strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thiếu tệp thoại hoặc mã luồng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendVoice(voiceURL, threadID, zaloThreadType(threadType), fileSize, 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi tin thoại: %w", err)
	}
	return nil
}

// SendSticker gửi nhãn dán.
func (c *Client) SendSticker(threadID string, threadType inbound.ThreadType,
	stickerType, stickerID, categoryID int) error {
	if strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thiếu mã luồng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendSticker(stickerType, stickerID, categoryID, threadID,
		zaloThreadType(threadType), 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi nhãn dán: %w", err)
	}
	return nil
}

// SendLink gửi một đường dẫn kèm lời nhắn, để Zalo tự dựng thẻ xem trước.
func (c *Client) SendLink(threadID string, threadType inbound.ThreadType, linkURL, caption string) error {
	if strings.TrimSpace(linkURL) == "" || strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thiếu đường dẫn hoặc mã luồng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	var message *zago.Message
	if strings.TrimSpace(caption) != "" {
		built := zago.Message{Text: caption}
		message = &built
	}
	if _, err = api.SendLink(linkURL, threadID, zaloThreadType(threadType), message, 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi đường dẫn: %w", err)
	}
	return nil
}

// SendMultiImage gửi nhiều ảnh trong một lượt.
//
// Zalo trả về MỘT kết quả cho MỖI ảnh; ở đây chỉ báo lỗi chung vì bên gọi
// không làm gì khác nhau với từng ảnh.
func (c *Client) SendMultiImage(threadID string, threadType inbound.ThreadType,
	imageURLs []string, caption string, width, height int) error {
	if len(imageURLs) == 0 || strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thiếu ảnh hoặc mã luồng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	var message *zago.Message
	if strings.TrimSpace(caption) != "" {
		built := zago.Message{Text: caption}
		message = &built
	}
	if _, err = api.SendMultiImage(imageURLs, threadID, zaloThreadType(threadType),
		width, height, message, 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi nhiều ảnh: %w", err)
	}
	return nil
}

// Quote là tin gốc đang được trả lời.
//
// Dựng lại từ những gì ĐÃ LƯU, không đọc lại Zalo: bên gọi giữ sẵn mã tin và
// chữ của tin gốc trong kho của mình.
type Quote struct {
	OwnerID  string
	MsgID    string
	CliMsgID string
	MsgType  string
	Ts       string
	Content  string
}

// SendQuote trả lời có trích dẫn một tin cũ.
func (c *Client) SendQuote(threadID string, threadType inbound.ThreadType, text string, quote Quote) error {
	if strings.TrimSpace(text) == "" || strings.TrimSpace(quote.MsgID) == "" {
		return errors.New("zalo-kit: thiếu nội dung hoặc tin gốc")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendMessageQuote(zago.Message{Text: text}, zago.QuoteInfo{
		OwnerID: quote.OwnerID, MsgID: quote.MsgID, CliMsgID: quote.CliMsgID,
		MsgType: quote.MsgType, Ts: quote.Ts, Content: quote.Content,
	}, threadID, zaloThreadType(threadType)); err != nil {
		return fmt.Errorf("zalo-kit: trả lời trích dẫn: %w", err)
	}
	return nil
}

// UndoMessage thu hồi một tin MÌNH đã gửi.
//
// ⚠️ Chỉ thu hồi được tin của chính nick này, và Zalo có hạn thời gian. Hết
// hạn thì trả lỗi — đừng nuốt lỗi đó rồi báo người dùng là đã thu hồi.
func (c *Client) UndoMessage(threadID string, threadType inbound.ThreadType, msgID, cliMsgID string) error {
	if strings.TrimSpace(msgID) == "" || strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thiếu mã tin hoặc mã luồng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.UndoMessage(msgID, cliMsgID, threadID, zaloThreadType(threadType)); err != nil {
		return fmt.Errorf("zalo-kit: thu hồi tin: %w", err)
	}
	return nil
}

// MarkAsRead báo cho Zalo là đã xem tin.
//
// Có mặt vì thiếu nó thì phía khách LUÔN thấy tin chưa được xem, dù nhân viên
// đã đọc và đã trả lời trong hộp thư của ta.
func (c *Client) MarkAsRead(threadID string, threadType inbound.ThreadType, msgID, cliMsgID, senderID string) error {
	return c.markSeen(threadID, threadType, msgID, cliMsgID, senderID, true)
}

// MarkAsDelivered báo cho Zalo là tin đã tới máy.
func (c *Client) MarkAsDelivered(threadID string, threadType inbound.ThreadType, msgID, cliMsgID, senderID string) error {
	return c.markSeen(threadID, threadType, msgID, cliMsgID, senderID, false)
}

func (c *Client) markSeen(threadID string, threadType inbound.ThreadType,
	msgID, cliMsgID, senderID string, read bool) error {
	if strings.TrimSpace(msgID) == "" || strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thiếu mã tin hoặc mã luồng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	goi := api.MarkAsDelivered
	viec := "đánh dấu đã nhận"
	if read {
		goi = api.MarkAsRead
		viec = "đánh dấu đã xem"
	}
	if _, err = goi(msgID, cliMsgID, senderID, threadID, zaloThreadType(threadType), ""); err != nil {
		return fmt.Errorf("zalo-kit: %s: %w", viec, err)
	}
	return nil
}
