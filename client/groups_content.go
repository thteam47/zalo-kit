package client

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thteam47/zago"
)

// Nội dung bên trong nhóm: bình chọn, tin ghim, ghi chú, bảng tin, và cài đặt.
//
// ⚠️ Các hàm ĐỌC ở đây trả map[string]any nguyên dạng Zalo, KHÔNG phải struct.
// Có chủ đích: ta chưa từng thấy dạng thật của những cây JSON này trên một
// phiên sống, và dựng struct từ phỏng đoán thì trường đoán sai bị bỏ ÂM THẦM —
// người đọc code tin rằng Zalo không trả trường đó. Trả map thì bên gọi thấy
// đúng những gì Zalo nói. Khi đã soi được dạng thật thì mới bọc thành struct.

// Poll là một cuộc bình chọn sắp tạo.
type Poll struct {
	Question          string
	Options           []string
	ExpiresAtUnix     int64
	Pin               bool
	MultiChoice       bool
	AllowNewOption    bool
	HideVoteUntilDone bool
	Anonymous         bool
}

// CreatePoll lập một cuộc bình chọn trong nhóm.
func (c *Client) CreatePoll(groupID string, poll Poll) error {
	question := strings.TrimSpace(poll.Question)
	options := locMa(poll.Options)
	if question == "" {
		return errors.New("zalo-kit: thiếu câu hỏi bình chọn")
	}
	// Zalo đòi ít nhất hai lựa chọn. Một lựa chọn thì nó trả lỗi mã số, và
	// locMa vừa bỏ trùng nên "A, A" cũng rơi xuống còn một.
	if len(options) < 2 {
		return errors.New("zalo-kit: bình chọn phải có ít nhất hai lựa chọn khác nhau")
	}
	return c.thaoTacNhom(groupID, "lập bình chọn", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.CreatePoll(question, options, strings.TrimSpace(groupID),
			poll.ExpiresAtUnix, poll.Pin, poll.MultiChoice, poll.AllowNewOption,
			poll.HideVoteUntilDone, poll.Anonymous)
	})
}

// VotePoll bỏ phiếu cho một hoặc nhiều lựa chọn.
func (c *Client) VotePoll(groupID string, pollID int64, optionIDs []string) error {
	options := locMa(optionIDs)
	if pollID == 0 || len(options) == 0 {
		return errors.New("zalo-kit: thiếu mã bình chọn hoặc lựa chọn")
	}
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return errors.New("zalo-kit: thiếu mã nhóm")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.VotePoll(pollID, options, groupID); err != nil {
		return fmt.Errorf("zalo-kit: bỏ phiếu: %w", err)
	}
	return nil
}

// LockPoll khoá bình chọn, không cho bỏ phiếu thêm.
func (c *Client) LockPoll(pollID int64) error {
	if pollID == 0 {
		return errors.New("zalo-kit: thiếu mã bình chọn")
	}
	return c.doiTuongNhom("khoá bình chọn", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.LockPoll(pollID)
	})
}

// PollDetail đọc chi tiết một cuộc bình chọn.
func (c *Client) PollDetail(pollID int64) (map[string]any, error) {
	if pollID == 0 {
		return nil, errors.New("zalo-kit: thiếu mã bình chọn")
	}
	return c.docNhom("đọc chi tiết bình chọn", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.ViewPollDetail(pollID)
	})
}

// Paging là vị trí đọc tiếp của các danh sách trong nhóm.
//
// Count = 0 thì dùng 20 — Zalo trả RỖNG chứ không trả mặc định khi count là 0,
// nên để nguyên 0 là một danh sách trống trông như "nhóm chưa có gì".
type Paging struct {
	Page     int
	Count    int
	LastID   int64
	LastType int
}

func (p Paging) dung() Paging {
	if p.Count <= 0 {
		p.Count = 20
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	return p
}

// GroupPolls đọc danh sách bình chọn của nhóm.
func (c *Client) GroupPolls(groupID string, paging Paging) (map[string]any, error) {
	return c.docNhomTheoTrang(groupID, paging, "đọc danh sách bình chọn",
		func(api *zago.ZaloAPI, id string, p Paging) (*zago.Group, error) {
			return api.GetGroupPoll(id, p.Page, p.Count, p.LastID, p.LastType)
		})
}

// GroupPinnedMessages đọc danh sách tin đang ghim.
func (c *Client) GroupPinnedMessages(groupID string, paging Paging) (map[string]any, error) {
	return c.docNhomTheoTrang(groupID, paging, "đọc tin ghim",
		func(api *zago.ZaloAPI, id string, p Paging) (*zago.Group, error) {
			return api.GetGroupPinMsg(id, p.Page, p.Count, p.LastID, p.LastType)
		})
}

// GroupNotes đọc ghi chú của nhóm.
func (c *Client) GroupNotes(groupID string, paging Paging) (map[string]any, error) {
	return c.docNhomTheoTrang(groupID, paging, "đọc ghi chú nhóm",
		func(api *zago.ZaloAPI, id string, p Paging) (*zago.Group, error) {
			return api.GetGroupNote(id, p.Page, p.Count, p.LastID, p.LastType)
		})
}

// GroupBoard đọc bảng tin của nhóm (gộp ghi chú, bình chọn, tin ghim).
func (c *Client) GroupBoard(groupID string, paging Paging) (map[string]any, error) {
	return c.docNhomTheoTrang(groupID, paging, "đọc bảng tin nhóm",
		func(api *zago.ZaloAPI, id string, p Paging) (*zago.Group, error) {
			return api.GetGroupBoardList(id, p.Page, p.Count, p.LastID, p.LastType)
		})
}

// SetGroupSetting đổi cài đặt nhóm.
//
// defaultMode và các khoá trong settings đi thẳng xuống Zalo. Không kiểm ở đây
// vì danh sách khoá của Zalo thay đổi theo phiên bản, và chặn theo danh sách
// cứng thì khoá mới bị từ chối oan.
func (c *Client) SetGroupSetting(groupID, defaultMode string, settings map[string]any) error {
	return c.thaoTacNhom(groupID, "đổi cài đặt nhóm", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.ChangeGroupSetting(strings.TrimSpace(groupID), strings.TrimSpace(defaultMode), settings)
	})
}

// UpgradeToCommunity nâng nhóm lên cộng đồng. Một chiều, KHÔNG hạ lại được.
func (c *Client) UpgradeToCommunity(groupID string) error {
	return c.thaoTacNhom(groupID, "nâng nhóm lên cộng đồng", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.UpgradeCommunity(strings.TrimSpace(groupID))
	})
}

// RegenerateGroupLink tạo lại đường mời, làm đường cũ hết hiệu lực.
//
// ⚠️ zago KHÔNG trả lỗi cho lời gọi này — cùng bẫy như GroupInviteLink, nên
// map rỗng cũng phải thành lỗi.
func (c *Client) RegenerateGroupLink(groupID string) (string, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return "", errors.New("zalo-kit: thiếu mã nhóm")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return "", err
	}
	link := timDuongMoi(api.GenerateNewLink(groupID), 0)
	if link == "" {
		return "", errors.New("zalo-kit: không tạo lại được đường mời nhóm")
	}
	return link, nil
}

// DisableGroupLink tắt đường mời của nhóm.
func (c *Client) DisableGroupLink(groupID string) error {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return errors.New("zalo-kit: thiếu mã nhóm")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	// zago không trả lỗi ở đây; map rỗng là dấu hiệu duy nhất có chuyện.
	if len(api.DisableLink(groupID)) == 0 {
		return errors.New("zalo-kit: không tắt được đường mời nhóm")
	}
	return nil
}

// InspectGroupLink xem một đường mời dẫn tới nhóm nào, trước khi vào.
func (c *Client) InspectGroupLink(inviteURL string) (map[string]any, error) {
	inviteURL = strings.TrimSpace(inviteURL)
	if inviteURL == "" {
		return nil, errors.New("zalo-kit: thiếu đường mời")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw := api.CheckGroup(inviteURL)
	if len(raw) == 0 {
		return nil, errors.New("zalo-kit: không đọc được đường mời nhóm")
	}
	return raw, nil
}

// BlockedGroupMembers đọc danh sách người bị chặn khỏi nhóm.
func (c *Client) BlockedGroupMembers(groupID string, paging Paging) (map[string]any, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, errors.New("zalo-kit: thiếu mã nhóm")
	}
	paging = paging.dung()
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw := api.GetBlockedMembers(groupID, paging.Page, paging.Count)
	if len(raw) == 0 {
		return nil, errors.New("zalo-kit: không đọc được danh sách chặn của nhóm")
	}
	return raw, nil
}

// GroupInviteBox đọc hộp lời mời vào nhóm.
func (c *Client) GroupInviteBox(paging Paging) (map[string]any, error) {
	paging = paging.dung()
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw, err := api.ListInviteBox(paging.Page, paging.Count, paging.Count, nil)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: đọc hộp lời mời nhóm: %w", err)
	}
	return thanhMap(raw), nil
}

// AcceptGroupInvite đồng ý một lời mời vào nhóm trong hộp lời mời.
func (c *Client) AcceptGroupInvite(groupID string) error {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return errors.New("zalo-kit: thiếu mã nhóm")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.BoxInviteAccept(groupID, "vi"); err != nil {
		return fmt.Errorf("zalo-kit: đồng ý lời mời vào nhóm: %w", err)
	}
	return nil
}

// RecentGroupActivity đọc hoạt động gần đây của một nhóm.
func (c *Client) RecentGroupActivity(groupID string) (map[string]any, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, errors.New("zalo-kit: thiếu mã nhóm")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return nil, err
	}
	raw, err := api.GetRecentGroup(groupID)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: đọc hoạt động gần đây của nhóm: %w", err)
	}
	return thanhMap(raw), nil
}

func (c *Client) docNhomTheoTrang(groupID string, paging Paging, viec string,
	goi func(*zago.ZaloAPI, string, Paging) (*zago.Group, error)) (map[string]any, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, errors.New("zalo-kit: thiếu mã nhóm")
	}
	paging = paging.dung()
	return c.docNhom(viec, func(api *zago.ZaloAPI) (*zago.Group, error) {
		return goi(api, groupID, paging)
	})
}

func (c *Client) docNhom(viec string, goi func(*zago.ZaloAPI) (*zago.Group, error)) (map[string]any, error) {
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
	// ⚠️ raw là con trỏ. zago trả nil kèm err == nil ở vài nhánh, và gọi
	// ToMap() trên con trỏ nil là panic giữa lúc đang chạy.
	if raw == nil {
		return map[string]any{}, nil
	}
	return raw.ToMap(), nil
}

// doiTuongNhom cho các lời gọi không gắn với một mã nhóm cụ thể.
func (c *Client) doiTuongNhom(viec string, goi func(*zago.ZaloAPI) (*zago.Group, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = goi(api); err != nil {
		return fmt.Errorf("zalo-kit: %s: %w", viec, err)
	}
	return nil
}

func thanhMap(value any) map[string]any {
	switch current := value.(type) {
	case nil:
		return map[string]any{}
	case map[string]any:
		return current
	case interface{ ToMap() map[string]any }:
		return current.ToMap()
	default:
		return map[string]any{"data": current}
	}
}
