package client

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

// Tài khoản của chính nick này, nhãn dán, và vài thao tác lẻ.

// AccountInfo đọc hồ sơ của chính nick đang đăng nhập.
//
// Trả map nguyên dạng vì Zalo nhét cả cấu hình doanh nghiệp, hạn mức, và cờ
// tính năng vào đây; cắt xuống một struct là vứt đi phần lớn.
func (c *Client) AccountInfo() (map[string]any, error) {
	return c.docThoTuTaiKhoan("đọc hồ sơ tài khoản", func(api *zago.ZaloAPI) (any, error) {
		return api.FetchAccountInfo()
	})
}

// SetAccountAvatar đổi ảnh đại diện của nick này.
//
// ⚠️ filePath là tệp TRÊN MÁY ĐANG CHẠY, không phải URL.
func (c *Client) SetAccountAvatar(filePath string, width, height int) error {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return errors.New("zalo-kit: thiếu đường dẫn ảnh")
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("zalo-kit: không đọc được ảnh: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.ChangeAccountAvatar(filePath, width, height, "vi", info.Size()); err != nil {
		return fmt.Errorf("zalo-kit: đổi ảnh đại diện: %w", err)
	}
	return nil
}

// AccountSetting là phần hồ sơ đổi được của chính nick này.
type AccountSetting struct {
	Name string
	// DOB dạng "2006-01-02".
	DOB string
	// Gender theo mã của Zalo. Không chuẩn hoá ở đây vì chiều NGƯỢC lại
	// (normalizeGender) cố ý trả rỗng khi không chắc — nếu ghi lại bằng giá trị
	// đã chuẩn hoá thì một lần đọc-rồi-ghi sẽ XOÁ giới tính của người dùng.
	Gender   int
	Business map[string]any
}

// SetAccountSetting đổi hồ sơ của nick này.
//
// ⚠️ Zalo GHI ĐÈ cả cụm, không vá từng trường. Truyền Name rỗng là xoá tên,
// nên bên gọi phải đọc hồ sơ hiện tại rồi sửa trên đó.
func (c *Client) SetAccountSetting(setting AccountSetting) error {
	if strings.TrimSpace(setting.Name) == "" {
		return errors.New("zalo-kit: thiếu tên hiển thị")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.ChangeAccountSetting(strings.TrimSpace(setting.Name),
		strings.TrimSpace(setting.DOB), setting.Gender, setting.Business, "vi"); err != nil {
		return fmt.Errorf("zalo-kit: đổi hồ sơ tài khoản: %w", err)
	}
	return nil
}

// Avatar đọc ảnh đại diện của một người ở kích thước mong muốn.
func (c *Client) Avatar(userID string, size int) (map[string]any, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("zalo-kit: thiếu mã người dùng")
	}
	if size <= 0 {
		size = 240
	}
	return c.docThoTuTaiKhoan("đọc ảnh đại diện", func(api *zago.ZaloAPI) (any, error) {
		return api.GetAvatar(userID, size)
	})
}

// LastMessages đọc tin cuối của từng hội thoại.
//
// Đây là thứ DUY NHẤT Zalo cho đọc lại khi vừa nối phiên: KHÔNG có API đọc
// toàn bộ lịch sử cũ. Nghĩa là hộp thư dựng được danh sách hội thoại kèm tin
// cuối ngay lần đầu vào, còn lịch sử đầy đủ chỉ lớn dần từ lúc ta bắt đầu
// lắng nghe. Đừng hứa với người dùng nhiều hơn thế.
func (c *Client) LastMessages() (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw, err := api.GetLastMsgs()
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: đọc tin cuối của các hội thoại: %w", err)
	}
	if raw == nil {
		return map[string]any{}, nil
	}
	return raw.ToMap(), nil
}

// SearchStickers tìm nhãn dán theo từ khoá.
func (c *Client) SearchStickers(keyword string, limit int) (map[string]any, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, errors.New("zalo-kit: thiếu từ khoá")
	}
	if limit <= 0 {
		limit = 20
	}
	return c.docThoTuTaiKhoan("tìm nhãn dán", func(api *zago.ZaloAPI) (any, error) {
		return api.SearchSticker(keyword, limit)
	})
}

// StickerDetail đọc chi tiết một nhãn dán.
func (c *Client) StickerDetail(stickerID int) (map[string]any, error) {
	if stickerID == 0 {
		return nil, errors.New("zalo-kit: thiếu mã nhãn dán")
	}
	return c.docThoTuTaiKhoan("đọc nhãn dán", func(api *zago.ZaloAPI) (any, error) {
		return api.GetSticker(stickerID)
	})
}

// StickerCategories đọc thông tin các bộ nhãn dán.
func (c *Client) StickerCategories(categoryIDs []string) (map[string]any, error) {
	ids := locMa(categoryIDs)
	if len(ids) == 0 {
		return nil, errors.New("zalo-kit: thiếu mã bộ nhãn dán")
	}
	return c.docThoTuTaiKhoan("đọc bộ nhãn dán", func(api *zago.ZaloAPI) (any, error) {
		return api.GetCategory(ids)
	})
}

// SendCustomSticker gửi nhãn dán tự tạo từ hai ảnh (tĩnh và động).
func (c *Client) SendCustomSticker(threadID string, threadType inbound.ThreadType,
	staticImageURL, animatedImageURL, reply string, width, height int) error {
	threadID, staticImageURL = strings.TrimSpace(threadID), strings.TrimSpace(staticImageURL)
	if threadID == "" || staticImageURL == "" {
		return errors.New("zalo-kit: thiếu mã luồng hoặc ảnh nhãn dán")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendCustomSticker(staticImageURL, strings.TrimSpace(animatedImageURL),
		threadID, zaloThreadType(threadType), strings.TrimSpace(reply),
		width, height, 0, false); err != nil {
		return fmt.Errorf("zalo-kit: gửi nhãn dán tự tạo: %w", err)
	}
	return nil
}

// SendBankCard gửi thẻ thông tin tài khoản ngân hàng.
//
// ⚠️ Tên ngân hàng phải là tên CHUẨN do Zalo nhận; gõ tắt hay viết tay thì thẻ
// hiện sai hoặc không hiện, và người chuyển tiền không biết mình chuyển đi đâu.
func (c *Client) SendBankCard(threadID string, threadType inbound.ThreadType,
	bankNumber, accountName, bankName string) error {
	threadID = strings.TrimSpace(threadID)
	bankNumber = strings.TrimSpace(bankNumber)
	accountName = strings.TrimSpace(accountName)
	bankName = strings.TrimSpace(bankName)
	if threadID == "" || bankNumber == "" || accountName == "" || bankName == "" {
		return errors.New("zalo-kit: thiếu mã luồng, số tài khoản, tên chủ tài khoản hoặc tên ngân hàng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendCardBank(nil, bankNumber, accountName, bankName,
		threadID, zaloThreadType(threadType)); err != nil {
		return fmt.Errorf("zalo-kit: gửi thẻ ngân hàng: %w", err)
	}
	return nil
}

// ReportUser báo xấu một người hoặc một nhóm.
func (c *Client) ReportUser(userID string, threadType inbound.ThreadType, reason int, content string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("zalo-kit: thiếu mã người dùng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendReport(userID, zaloThreadType(threadType), reason,
		strings.TrimSpace(content)); err != nil {
		return fmt.Errorf("zalo-kit: báo xấu: %w", err)
	}
	return nil
}

// SetFeedBlocked ẩn hoặc hiện lại nhật ký của một người.
//
// Khác BlockUser: chặn là cắt cả tin nhắn, còn cái này chỉ ẩn nhật ký.
func (c *Client) SetFeedBlocked(userID string, blocked bool) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("zalo-kit: thiếu mã người dùng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.BlockViewFeed(userID, blocked); err != nil {
		return fmt.Errorf("zalo-kit: đổi chế độ xem nhật ký: %w", err)
	}
	return nil
}

func (c *Client) docThoTuTaiKhoan(viec string, goi func(*zago.ZaloAPI) (any, error)) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw, err := goi(api)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: %s: %w", viec, err)
	}
	return thanhMap(raw), nil
}
