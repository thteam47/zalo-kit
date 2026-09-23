package client

import (
	"errors"
	"fmt"
	"strings"
)

// ErrContactNotFound: số đó không có tài khoản Zalo, hoặc chủ số đặt riêng tư.
//
// Tách hẳn khỏi lỗi mạng/phiên có chủ đích. Hai chuyện này dẫn tới hai việc
// khác nhau của người dùng: "số này chưa có Zalo, nhập số khác" so với "mạng
// đang lỗi, thử lại". Gộp thành một câu là người ta đi sửa nhầm chỗ.
var ErrContactNotFound = errors.New("zalo-kit: không tìm thấy tài khoản Zalo cho số này")

// FindByPhone tra một số điện thoại thành hồ sơ Zalo.
//
// KHÔNG tự chuẩn hoá số ở đây: zago đã có util.NormalizePhone chạy ngay trước
// khi gọi Zalo. Chuẩn hoá thêm một lần nữa ở tầng này là hai bộ luật cùng sửa
// một chuỗi, và khi Zalo đổi định dạng thì phải nhớ sửa cả hai chỗ.
func (c *Client) FindByPhone(phone string) (Profile, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return Profile{}, errors.New("zalo-kit: thiếu số điện thoại")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.api == nil {
		return Profile{}, ErrSessionInvalid
	}
	raw, err := c.api.FetchPhoneNumber(phone, "vi")
	if err != nil {
		return Profile{}, fmt.Errorf("zalo-kit: tra số điện thoại trên Zalo: %w", err)
	}

	// Dùng lại đúng bộ nhặt hồ sơ của FetchProfiles: Zalo trả về nhiều dạng
	// lồng nhau tuỳ endpoint, và bộ đó đã chịu được cả ba dạng đã gặp.
	found := map[string]Profile{}
	collectProfiles(raw, found, 0)
	for _, profile := range found {
		if strings.TrimSpace(profile.UserID) != "" {
			return profile, nil
		}
	}
	return Profile{}, ErrContactNotFound
}

// AddFriend gửi lời mời kết bạn tới một uid.
//
// ⚠️ Zalo trả về 200 cho nhiều trường hợp KHÔNG phải thành công: đã là bạn,
// đã gửi lời mời trước đó, người kia chặn nhận lời mời. Lớp này chỉ báo lại
// lỗi TRUYỀN TẢI — bên gọi đừng đọc "không lỗi" thành "chắc chắn đã gửi".
//
// Không tự thử lại: gửi lời mời hai lần là một hành vi nhìn thấy được ở phía
// người nhận, và nhiều lời mời trong thời gian ngắn là đúng thứ làm nick bị
// hạn chế.
func (c *Client) AddFriend(userID, message string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("zalo-kit: thiếu mã người nhận")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.api == nil {
		return ErrSessionInvalid
	}
	if _, err := c.api.AddFriend(userID, strings.TrimSpace(message), "vi"); err != nil {
		return fmt.Errorf("zalo-kit: gửi lời mời kết bạn: %w", err)
	}
	return nil
}
