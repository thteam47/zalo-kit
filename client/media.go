package client

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

// Gửi tệp nằm TRÊN ĐĨA của máy đang chạy.
//
// Khác hẳn send_more.go: ở đó Zalo tự đi lấy theo URL, nên đường ký tạm hết
// hạn là hỏng ngẫu nhiên. Ở đây ta tải byte lên trước rồi mới gửi, nên dùng
// được cho tệp người dùng vừa chọn — đúng luồng của một ứng dụng chat.

// Upload là kết quả tải một tệp lên kho của Zalo.
type Upload struct {
	// URL là đường Zalo dùng lại được. Rỗng nghĩa là KHÔNG gửi được — bên gọi
	// phải coi đó là lỗi chứ đừng gửi tiếp với chuỗi rỗng.
	URL      string
	FileName string
	FileSize int
	Raw      map[string]any
}

// UploadFile tải một tệp trên đĩa lên kho của Zalo mà chưa gửi.
//
// Tách khỏi việc gửi vì hai lý do có thật: gửi cùng một tệp vào nhiều luồng
// thì chỉ tải một lần, và giao diện cần hiện thanh tiến trình tải xong trước
// khi tin xuất hiện trong hội thoại.
func (c *Client) UploadFile(threadID string, threadType inbound.ThreadType, filePath string) (Upload, error) {
	return c.taiLen(threadID, threadType, filePath, false)
}

// UploadImage tải một ảnh trên đĩa lên kho của Zalo.
func (c *Client) UploadImage(threadID string, threadType inbound.ThreadType, filePath string) (Upload, error) {
	return c.taiLen(threadID, threadType, filePath, true)
}

func (c *Client) taiLen(threadID string, threadType inbound.ThreadType, filePath string, laAnh bool) (Upload, error) {
	threadID, filePath = strings.TrimSpace(threadID), strings.TrimSpace(filePath)
	if threadID == "" || filePath == "" {
		return Upload{}, errors.New("zalo-kit: thiếu mã luồng hoặc đường dẫn tệp")
	}
	// Kiểm tệp TRƯỚC khi khoá và trước khi đụng tới mạng: đường dẫn sai là lỗi
	// phổ biến nhất ở đây, và báo "không mở được tệp" hữu ích hơn nhiều so với
	// một lỗi HTTP của Zalo.
	info, err := os.Stat(filePath)
	if err != nil {
		return Upload{}, fmt.Errorf("zalo-kit: không đọc được tệp: %w", err)
	}
	if info.IsDir() {
		return Upload{}, errors.New("zalo-kit: đường dẫn là thư mục, không phải tệp")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return Upload{}, err
	}
	goi := api.UploadAttachment
	viec := "tải tệp lên"
	if laAnh {
		goi = api.UploadImage
		viec = "tải ảnh lên"
	}
	raw, err := goi(filePath, threadID, zaloThreadType(threadType))
	if err != nil {
		return Upload{}, fmt.Errorf("zalo-kit: %s: %w", viec, err)
	}
	up := Upload{
		URL:      timDuongTep(raw, 0),
		FileName: filepath.Base(filePath),
		FileSize: int(info.Size()),
		Raw:      raw,
	}
	if up.URL == "" {
		return up, errors.New("zalo-kit: tải lên xong nhưng không đọc được đường dẫn tệp")
	}
	return up, nil
}

// SendLocalFile tải tệp lên rồi gửi trong một lượt.
func (c *Client) SendLocalFile(threadID string, threadType inbound.ThreadType, filePath string) error {
	up, err := c.UploadFile(threadID, threadType, filePath)
	if err != nil {
		return err
	}
	ext := strings.TrimPrefix(filepath.Ext(up.FileName), ".")
	return c.SendFile(threadID, threadType, up.URL, up.FileName, ext, up.FileSize)
}

// SendLocalImage gửi một ảnh trên đĩa kèm lời nhắn.
//
// width/height để 0 thì Zalo tự đoán; sai kích thước chỉ làm ảnh xem trước bị
// méo chứ không mất ảnh.
func (c *Client) SendLocalImage(threadID string, threadType inbound.ThreadType,
	imagePath, caption string, width, height int) error {
	threadID, imagePath = strings.TrimSpace(threadID), strings.TrimSpace(imagePath)
	if threadID == "" || imagePath == "" {
		return errors.New("zalo-kit: thiếu mã luồng hoặc đường dẫn ảnh")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendLocalImage(imagePath, threadID, zaloThreadType(threadType),
		width, height, loiNhan(caption), nil, 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi ảnh từ máy: %w", err)
	}
	return nil
}

// SendLocalImages gửi nhiều ảnh trên đĩa trong một lượt.
func (c *Client) SendLocalImages(threadID string, threadType inbound.ThreadType,
	imagePaths []string, caption string, width, height int) error {
	paths := locMa(imagePaths)
	if len(paths) == 0 || strings.TrimSpace(threadID) == "" {
		return errors.New("zalo-kit: thiếu ảnh hoặc mã luồng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendMultiLocalImage(paths, strings.TrimSpace(threadID),
		zaloThreadType(threadType), width, height, loiNhan(caption), 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi nhiều ảnh từ máy: %w", err)
	}
	return nil
}

// SendLocalGif gửi ảnh động từ máy.
func (c *Client) SendLocalGif(threadID string, threadType inbound.ThreadType,
	gifPath, thumbnailURL, gifName string, width, height int) error {
	threadID, gifPath = strings.TrimSpace(threadID), strings.TrimSpace(gifPath)
	if threadID == "" || gifPath == "" {
		return errors.New("zalo-kit: thiếu mã luồng hoặc đường dẫn ảnh động")
	}
	if strings.TrimSpace(gifName) == "" {
		gifName = filepath.Base(gifPath)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendLocalGif(gifPath, thumbnailURL, threadID,
		zaloThreadType(threadType), gifName, width, height, 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi ảnh động: %w", err)
	}
	return nil
}

// SendBusinessCard gửi danh thiếp của một người trên Zalo.
func (c *Client) SendBusinessCard(threadID string, threadType inbound.ThreadType,
	userID, phone, qrCodeURL string) error {
	threadID, userID = strings.TrimSpace(threadID), strings.TrimSpace(userID)
	if threadID == "" || userID == "" {
		return errors.New("zalo-kit: thiếu mã luồng hoặc mã người dùng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendBusinessCard(userID, strings.TrimSpace(qrCodeURL), threadID,
		zaloThreadType(threadType), strings.TrimSpace(phone), 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi danh thiếp: %w", err)
	}
	return nil
}

// SetAutoDeleteChat đặt hạn tự xoá tin của một hội thoại, tính bằng giây.
//
// ttl = 0 là TẮT tự xoá. Đây là cài đặt của Zalo, không đụng tới kho của ta —
// tin đã lưu bên mình vẫn còn.
func (c *Client) SetAutoDeleteChat(threadID string, threadType inbound.ThreadType, ttlSeconds int) error {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return errors.New("zalo-kit: thiếu mã luồng")
	}
	if ttlSeconds < 0 {
		return errors.New("zalo-kit: hạn tự xoá không được âm")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.UpdateAutoDeleteChat(ttlSeconds, threadID, zaloThreadType(threadType)); err != nil {
		return fmt.Errorf("zalo-kit: đặt hạn tự xoá tin: %w", err)
	}
	return nil
}

// loiNhan đổi lời nhắn rỗng thành nil — zago hiểu nil là "không có chú thích",
// còn một Message rỗng là một tin chữ TRỐNG gửi kèm.
func loiNhan(caption string) *zago.Message {
	caption = strings.TrimSpace(caption)
	if caption == "" {
		return nil
	}
	built := zago.Message{Text: caption}
	return &built
}

// timDuongTep nhặt đường dẫn tệp trong cây trả về của các lời gọi tải lên.
//
// Zalo đặt tên khoá khác nhau tuỳ loại tệp (ảnh có normalUrl/hdUrl, tệp thường
// có fileUrl), nên duyệt cả cây thay vì đoán một khoá.
func timDuongTep(value any, depth int) string {
	if depth > 6 {
		return ""
	}
	switch current := value.(type) {
	case map[string]any:
		if url := firstString(current, "fileUrl", "normalUrl", "hdUrl", "url", "href", "file_url"); url != "" {
			return url
		}
		for _, nested := range current {
			if url := timDuongTep(nested, depth+1); url != "" {
				return url
			}
		}
	case []any:
		for _, nested := range current {
			if url := timDuongTep(nested, depth+1); url != "" {
				return url
			}
		}
	case interface{ ToMap() map[string]any }:
		return timDuongTep(current.ToMap(), depth+1)
	}
	return ""
}
