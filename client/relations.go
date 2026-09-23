package client

import (
	"errors"
	"fmt"
	"strings"
)

// Quan hệ với người khác: kết bạn, huỷ bạn, chặn, đặt tên gợi nhớ, và đọc
// danh bạ.
//
// Mọi hàm trả danh sách đều trả []Profile chứ không phải `any`. Zalo trả về
// cây JSON nhiều tầng khác nhau tuỳ endpoint; để `any` chạy ra ngoài nghĩa là
// mỗi nơi gọi lại tự đoán một dạng, và nơi nào đoán sai thì im lặng trả rỗng.

// AcceptFriendRequest đồng ý lời mời kết bạn của một người.
func (c *Client) AcceptFriendRequest(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("zalo-kit: thiếu mã người gửi lời mời")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.AcceptFriendRequest(userID, "vi"); err != nil {
		return fmt.Errorf("zalo-kit: đồng ý kết bạn: %w", err)
	}
	return nil
}

// UnfriendUser huỷ kết bạn.
func (c *Client) UnfriendUser(userID string) error {
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
	if _, err = api.UnfriendUser(userID, "vi"); err != nil {
		return fmt.Errorf("zalo-kit: huỷ kết bạn: %w", err)
	}
	return nil
}

// BlockUser chặn một người.
func (c *Client) BlockUser(userID string) error { return c.setBlock(userID, true) }

// UnblockUser bỏ chặn.
func (c *Client) UnblockUser(userID string) error { return c.setBlock(userID, false) }

func (c *Client) setBlock(userID string, block bool) error {
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
	goi, viec := api.UnblockUser, "bỏ chặn"
	if block {
		goi, viec = api.BlockUser, "chặn"
	}
	if _, err = goi(userID); err != nil {
		return fmt.Errorf("zalo-kit: %s người dùng: %w", viec, err)
	}
	return nil
}

// SetAlias đặt tên gợi nhớ cho một người.
//
// Tên gợi nhớ chỉ nick NÀY thấy, không đổi tên hiển thị của người ta. Đặt
// alias rỗng là XOÁ tên gợi nhớ — đó là cách Zalo làm, không phải lỗi.
func (c *Client) SetAlias(userID, alias string) error {
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
	if _, err = api.SetAlias(userID, strings.TrimSpace(alias)); err != nil {
		return fmt.Errorf("zalo-kit: đặt tên gợi nhớ: %w", err)
	}
	return nil
}

// ListFriends đọc toàn bộ danh bạ bạn bè của nick này.
func (c *Client) ListFriends() ([]Profile, error) {
	return c.danhSach(func() (any, error) { return c.api.FetchAllFriends() }, "đọc danh bạ")
}

// ListFriendRequests đọc lời mời kết bạn ĐẾN.
func (c *Client) ListFriendRequests() ([]Profile, error) {
	return c.danhSach(func() (any, error) { return c.api.ListFriendRequests() }, "đọc lời mời đến")
}

// ListSentFriendRequests đọc lời mời kết bạn MÌNH ĐÃ GỬI.
//
// Cần để biết đã mời ai rồi mà chưa được đồng ý — gửi lại lần nữa là hành vi
// nhìn thấy được ở phía người nhận.
func (c *Client) ListSentFriendRequests() ([]Profile, error) {
	return c.danhSach(func() (any, error) { return c.api.ListSentFriendRequests() }, "đọc lời mời đã gửi")
}

// danhSach gom đúng khuôn: khoá, kiểm phiên, gọi, rồi nhặt hồ sơ khỏi cây JSON.
func (c *Client) danhSach(goi func() (any, error), viec string) ([]Profile, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.guard(); err != nil {
		return nil, err
	}
	raw, err := goi()
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: %s: %w", viec, err)
	}
	found := map[string]Profile{}
	collectProfiles(raw, found, 0)
	out := make([]Profile, 0, len(found))
	for _, profile := range found {
		out = append(out, profile)
	}
	return out, nil
}
