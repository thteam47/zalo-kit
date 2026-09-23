package client

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

// Phần còn lại: gọi thoại, ảnh động, cảm xúc nhiều lần, và mấy ô trạng thái.

// IsReady cho biết đường truyền đã sẵn sàng gửi chưa.
//
// Khác IsLoggedIn: có cookie hợp lệ mà ổ cắm chưa nối thì vẫn đăng nhập nhưng
// KHÔNG gửi được. Hộp thư phải phân biệt được hai cái để hiện đúng trạng thái
// thay vì để người dùng gõ vào một ô không gửi đi đâu cả.
func (c *Client) IsReady() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.api != nil && c.api.IsReady()
}

// IsListening cho biết vòng nhận tin có đang chạy không.
func (c *Client) IsListening() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.api != nil && c.api.IsListening()
}

// ClientID là mã máy khách Zalo cấp cho phiên này.
func (c *Client) ClientID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.api == nil {
		return ""
	}
	return c.api.ClientID()
}

// ProxyURL là proxy phiên này đang đi qua. Rỗng là đi thẳng.
func (c *Client) ProxyURL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.api == nil {
		return ""
	}
	return c.api.ProxyURL()
}

// QRLink lấy đường mã QR danh thiếp của một người.
func (c *Client) QRLink(userID string) (map[string]any, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("zalo-kit: thiếu mã người dùng")
	}
	return c.docThoTuTaiKhoan("đọc mã QR danh thiếp", func(api *zago.ZaloAPI) (any, error) {
		return api.GetQRLink(userID)
	})
}

// FindSticker tìm nhãn dán theo từ khoá, dạng gợi ý nhanh.
//
// Khác SearchStickers ở chỗ không nhận giới hạn số lượng — đây là hai endpoint
// khác nhau của Zalo chứ không phải một cái bọc hai lần.
func (c *Client) FindSticker(keyword string) (map[string]any, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, errors.New("zalo-kit: thiếu từ khoá")
	}
	return c.docThoTuTaiKhoan("tìm nhanh nhãn dán", func(api *zago.ZaloAPI) (any, error) {
		return api.FindSticker(keyword)
	})
}

// SyncPersonalStickers đồng bộ bộ nhãn dán cá nhân.
func (c *Client) SyncPersonalStickers(categoryIDs []string, version int) error {
	ids := locMa(categoryIDs)
	if len(ids) == 0 {
		return errors.New("zalo-kit: thiếu mã bộ nhãn dán")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.UpdatePersonalSticker(ids, version); err != nil {
		return fmt.Errorf("zalo-kit: đồng bộ nhãn dán cá nhân: %w", err)
	}
	return nil
}

// SendGif gửi ảnh động lên kho của Zalo rồi gửi.
//
// gifPath là tệp TRÊN MÁY ĐANG CHẠY. Khác SendLocalGif ở endpoint Zalo dùng,
// không phải ở phía ta.
func (c *Client) SendGif(threadID string, threadType inbound.ThreadType,
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
	if _, err = api.SendGifphy(gifPath, thumbnailURL, threadID,
		zaloThreadType(threadType), gifName, width, height, 0); err != nil {
		return fmt.Errorf("zalo-kit: gửi ảnh động: %w", err)
	}
	return nil
}

// SendMultiReaction thả cùng một cảm xúc nhiều lần lên một tin.
//
// count phải dương: Zalo nhận 0 rồi KHÔNG làm gì và vẫn trả 200, nên bên gọi
// tưởng đã thả mà không có gì xuất hiện.
func (c *Client) SendMultiReaction(threadID string, threadType inbound.ThreadType,
	target MessageRef, icon string, count int) error {
	threadID, icon = strings.TrimSpace(threadID), strings.TrimSpace(icon)
	if threadID == "" || strings.TrimSpace(target.MsgID) == "" || icon == "" {
		return errors.New("zalo-kit: thiếu mã luồng, mã tin hoặc biểu tượng")
	}
	if count <= 0 {
		return errors.New("zalo-kit: số lần thả cảm xúc phải lớn hơn 0")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendMultiReaction(target.toMessageObject(), icon, threadID,
		zaloThreadType(threadType), 0, count); err != nil {
		return fmt.Errorf("zalo-kit: thả cảm xúc nhiều lần: %w", err)
	}
	return nil
}

// GroupInfoV2 đọc thông tin nhóm bằng endpoint đời mới.
//
// payload đi thẳng xuống Zalo vì cấu trúc của nó thay đổi theo phiên bản; chặn
// theo danh sách khoá cứng thì khoá mới bị từ chối oan.
func (c *Client) GroupInfoV2(payload map[string]any) (map[string]any, error) {
	if len(payload) == 0 {
		return nil, errors.New("zalo-kit: thiếu nội dung yêu cầu")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw, err := api.GetGroupInfoV2(payload)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: đọc thông tin nhóm (v2): %w", err)
	}
	return raw, nil
}

// Gọi thoại và video.
//
// ⚠️ zalo-kit CHỈ báo hiệu, KHÔNG truyền tiếng hay hình. Những lời gọi dưới
// đây dựng và huỷ một cuộc gọi ở phía Zalo; muốn thật sự nghe nói thì phải có
// tầng media riêng. Đừng bọc chúng ra giao diện như một nút "gọi" hoàn chỉnh.

// StartCall bắt đầu một cuộc gọi tới một người.
func (c *Client) StartCall(targetID, callID string) (map[string]any, error) {
	targetID, callID = strings.TrimSpace(targetID), strings.TrimSpace(callID)
	if targetID == "" || callID == "" {
		return nil, errors.New("zalo-kit: thiếu mã người nhận hoặc mã cuộc gọi")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw, err := api.SendCall(targetID, callID)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: bắt đầu cuộc gọi: %w", err)
	}
	return raw, nil
}

// StartGroupCall bắt đầu cuộc gọi nhóm.
func (c *Client) StartGroupCall(groupID string, userIDs []string) (map[string]any, error) {
	groupID = strings.TrimSpace(groupID)
	members := locMa(userIDs)
	if groupID == "" || len(members) == 0 {
		return nil, errors.New("zalo-kit: thiếu mã nhóm hoặc người tham gia")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw, err := api.CallGroup(groupID, members)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: bắt đầu cuộc gọi nhóm: %w", err)
	}
	return raw, nil
}

// InviteToGroupCall mời thêm người vào cuộc gọi nhóm đang diễn ra.
func (c *Client) InviteToGroupCall(groupID, callID, hostCallID string, userIDs []string) (map[string]any, error) {
	groupID, callID = strings.TrimSpace(groupID), strings.TrimSpace(callID)
	members := locMa(userIDs)
	if groupID == "" || callID == "" || len(members) == 0 {
		return nil, errors.New("zalo-kit: thiếu mã nhóm, mã cuộc gọi hoặc người được mời")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw, err := api.CallGroupAdd(members, callID, strings.TrimSpace(hostCallID), groupID)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: mời thêm vào cuộc gọi nhóm: %w", err)
	}
	return raw, nil
}

// RequestGroupCall xin vào một cuộc gọi nhóm.
func (c *Client) RequestGroupCall(groupID, callID string, userIDs []string) (map[string]any, error) {
	groupID, callID = strings.TrimSpace(groupID), strings.TrimSpace(callID)
	members := locMa(userIDs)
	if groupID == "" || callID == "" || len(members) == 0 {
		return nil, errors.New("zalo-kit: thiếu mã nhóm, mã cuộc gọi hoặc người tham gia")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw, err := api.CallGroupRequest(groupID, members, callID)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: xin vào cuộc gọi nhóm: %w", err)
	}
	return raw, nil
}

// CancelGroupCall huỷ cuộc gọi nhóm.
func (c *Client) CancelGroupCall(groupID, callID, hostCallID string) error {
	groupID, callID = strings.TrimSpace(groupID), strings.TrimSpace(callID)
	if groupID == "" || callID == "" {
		return errors.New("zalo-kit: thiếu mã nhóm hoặc mã cuộc gọi")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.CallGroupCancel(callID, strings.TrimSpace(hostCallID), groupID); err != nil {
		return fmt.Errorf("zalo-kit: huỷ cuộc gọi nhóm: %w", err)
	}
	return nil
}
