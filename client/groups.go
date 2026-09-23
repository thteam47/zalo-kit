package client

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

// Nhóm Zalo: đọc danh sách, thành viên, và các thao tác quản trị.
//
// Một ứng dụng chat không thể chỉ nhắn tin một-một. Thiếu lớp này thì mọi
// luồng nhóm — vốn là phần lớn hội thoại thật — chỉ đọc được tin đến mà không
// làm được gì với nhóm.

// Group là nhóm Zalo ở dạng ổn định của ta.
type Group struct {
	GroupID     string
	Name        string
	Avatar      string
	OwnerID     string
	MemberCount int
}

// ListGroups đọc mọi nhóm nick này đang ở trong.
func (c *Client) ListGroups() ([]Group, error) {
	return c.nhomTheoCay(func() (any, error) { return c.api.FetchAllGroups() }, "đọc danh sách nhóm")
}

// GroupInfo đọc thông tin chi tiết của một hoặc nhiều nhóm.
func (c *Client) GroupInfo(groupIDs ...string) ([]Group, error) {
	ids := locMa(groupIDs)
	if len(ids) == 0 {
		return nil, errors.New("zalo-kit: thiếu mã nhóm")
	}
	return c.nhomTheoCay(func() (any, error) { return c.api.FetchGroupInfo(ids...) }, "đọc thông tin nhóm")
}

// GroupMembers đọc thành viên của một nhóm.
//
// Trả []Profile chứ không phải danh sách uid trần: hộp thư cần TÊN để hiện ai
// đang nói trong nhóm, và gọi tra tên riêng cho từng uid là thêm một vòng
// mạng cho mỗi tin nhóm.
func (c *Client) GroupMembers(groupID string) ([]Profile, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, errors.New("zalo-kit: thiếu mã nhóm")
	}
	return c.danhSach(func() (any, error) { return c.api.GetGroupMember(groupID) }, "đọc thành viên nhóm")
}

// CreateGroup lập nhóm mới và trả về nhóm vừa lập.
//
// ⚠️ Gọi với danh sách thành viên rỗng thì Zalo trả lỗi mã số khó đọc, nên
// chặn sớm ở đây.
func (c *Client) CreateGroup(name, description string, memberIDs []string) (Group, error) {
	name = strings.TrimSpace(name)
	members := locMa(memberIDs)
	if name == "" {
		return Group{}, errors.New("zalo-kit: thiếu tên nhóm")
	}
	if len(members) == 0 {
		return Group{}, errors.New("zalo-kit: phải có ít nhất một thành viên")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return Group{}, err
	}
	raw, err := api.CreateGroup(name, strings.TrimSpace(description), members, 1, 1)
	if err != nil {
		return Group{}, fmt.Errorf("zalo-kit: lập nhóm: %w", err)
	}
	found := map[string]Group{}
	collectGroups(raw, found, 0)
	for _, group := range found {
		return group, nil
	}
	// Zalo có lúc chỉ trả mã nhóm trần chứ không kèm tên, nên groupFromMap
	// không nhận ra. Không đọc được mã nghĩa là bên gọi không nhắn được vào
	// nhóm vừa lập — phải báo lỗi chứ đừng trả rỗng im lặng.
	if id := maNhomTran(raw, 0); id != "" {
		return Group{GroupID: id, Name: name}, nil
	}
	return Group{}, errors.New("zalo-kit: lập nhóm xong nhưng không đọc được mã nhóm")
}

// AddGroupMembers mời thêm người vào nhóm.
func (c *Client) AddGroupMembers(groupID string, userIDs []string) error {
	return c.thaoTacThanhVien(groupID, userIDs, "thêm thành viên",
		func(api *zago.ZaloAPI, members any, id string) (*zago.Group, error) {
			return api.AddUsersToGroup(members, id)
		})
}

// RemoveGroupMembers mời ra khỏi nhóm. Cần quyền quản trị.
func (c *Client) RemoveGroupMembers(groupID string, userIDs []string) error {
	return c.thaoTacThanhVien(groupID, userIDs, "mời ra khỏi nhóm",
		func(api *zago.ZaloAPI, members any, id string) (*zago.Group, error) {
			return api.KickUsers(members, id)
		})
}

// AddGroupAdmins trao quyền quản trị.
func (c *Client) AddGroupAdmins(groupID string, userIDs []string) error {
	return c.thaoTacThanhVien(groupID, userIDs, "trao quyền quản trị",
		func(api *zago.ZaloAPI, members any, id string) (*zago.Group, error) {
			return api.AddAdmins(members, id)
		})
}

// RemoveGroupAdmins thu quyền quản trị.
func (c *Client) RemoveGroupAdmins(groupID string, userIDs []string) error {
	return c.thaoTacThanhVien(groupID, userIDs, "thu quyền quản trị",
		func(api *zago.ZaloAPI, members any, id string) (*zago.Group, error) {
			return api.RemoveAdmins(members, id)
		})
}

// BlockGroupMembers chặn người khỏi vào lại nhóm.
func (c *Client) BlockGroupMembers(groupID string, userIDs []string) error {
	return c.thaoTacThanhVien(groupID, userIDs, "chặn khỏi nhóm",
		func(api *zago.ZaloAPI, members any, id string) (*zago.Group, error) {
			return api.BlockUsers(members, id)
		})
}

// UnblockGroupMembers bỏ chặn khỏi nhóm.
func (c *Client) UnblockGroupMembers(groupID string, userIDs []string) error {
	return c.thaoTacThanhVien(groupID, userIDs, "bỏ chặn khỏi nhóm",
		func(api *zago.ZaloAPI, members any, id string) (*zago.Group, error) {
			return api.UnblockUsers(members, id)
		})
}

// TransferGroupOwner chuyển quyền chủ nhóm.
//
// ⚠️ Một chiều. Chuyển xong thì nick này KHÔNG tự lấy lại được.
func (c *Client) TransferGroupOwner(groupID, newOwnerID string) error {
	newOwnerID = strings.TrimSpace(newOwnerID)
	if newOwnerID == "" {
		return errors.New("zalo-kit: thiếu mã chủ nhóm mới")
	}
	return c.thaoTacNhom(groupID, "chuyển quyền chủ nhóm", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.ChangeGroupOwner(newOwnerID, strings.TrimSpace(groupID))
	})
}

// RenameGroup đổi tên nhóm.
func (c *Client) RenameGroup(groupID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("zalo-kit: thiếu tên nhóm mới")
	}
	return c.thaoTacNhom(groupID, "đổi tên nhóm", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.ChangeGroupName(name, strings.TrimSpace(groupID))
	})
}

// SetGroupAvatar đổi ảnh đại diện nhóm.
//
// ⚠️ filePath là đường dẫn TỆP TRÊN MÁY ĐANG CHẠY, không phải URL. Chạy trên
// máy chủ thì phải tải ảnh về đĩa trước khi gọi.
func (c *Client) SetGroupAvatar(groupID, filePath string) error {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return errors.New("zalo-kit: thiếu đường dẫn ảnh")
	}
	return c.thaoTacNhom(groupID, "đổi ảnh nhóm", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.ChangeGroupAvatar(filePath, strings.TrimSpace(groupID))
	})
}

// DisbandGroup giải tán nhóm. Chỉ chủ nhóm làm được, và KHÔNG hoàn tác được.
func (c *Client) DisbandGroup(groupID string) error {
	return c.thaoTacNhom(groupID, "giải tán nhóm", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.DisperseGroup(strings.TrimSpace(groupID))
	})
}

// LeaveGroup rời nhóm. silent = rời mà không báo cho nhóm biết.
func (c *Client) LeaveGroup(groupID string, silent bool) error {
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
	if _, err = api.LeaveGroup(groupID, silent); err != nil {
		return fmt.Errorf("zalo-kit: rời nhóm: %w", err)
	}
	return nil
}

// JoinGroupByLink vào nhóm bằng đường mời.
func (c *Client) JoinGroupByLink(inviteURL string) error {
	inviteURL = strings.TrimSpace(inviteURL)
	if inviteURL == "" {
		return errors.New("zalo-kit: thiếu đường mời")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.JoinGroup(inviteURL); err != nil {
		return fmt.Errorf("zalo-kit: vào nhóm theo đường mời: %w", err)
	}
	return nil
}

// GroupInviteLink lấy đường mời vào nhóm.
//
// ⚠️ zago KHÔNG trả lỗi cho lời gọi này, chỉ trả map — mạng hỏng cũng ra map
// rỗng. Ở đây đổi map rỗng thành lỗi, vì bên gọi dán một chuỗi rỗng vào tin
// nhắn thì khách nhận được lời mời không có đường dẫn.
func (c *Client) GroupInviteLink(groupID string) (string, error) {
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
	link := timDuongMoi(api.GetGroupLink(groupID), 0)
	if link == "" {
		return "", errors.New("zalo-kit: không lấy được đường mời nhóm")
	}
	return link, nil
}

// MuteGroup tắt hoặc bật lại thông báo của một nhóm.
func (c *Client) MuteGroup(groupID string, mute bool) error {
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
	if _, err = api.SetMute(groupID, mute); err != nil {
		return fmt.Errorf("zalo-kit: đổi chế độ thông báo nhóm: %w", err)
	}
	return nil
}

// PendingMembers đọc danh sách người đang chờ duyệt vào nhóm.
func (c *Client) PendingMembers(groupID string) ([]Profile, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, errors.New("zalo-kit: thiếu mã nhóm")
	}
	return c.danhSach(func() (any, error) { return c.api.ViewGroupPending(groupID) }, "đọc danh sách chờ duyệt")
}

// ReviewPendingMembers duyệt hoặc từ chối người đang chờ vào nhóm.
func (c *Client) ReviewPendingMembers(groupID string, userIDs []string, approve bool) error {
	viec := "từ chối vào nhóm"
	if approve {
		viec = "duyệt vào nhóm"
	}
	return c.thaoTacThanhVien(groupID, userIDs, viec,
		func(api *zago.ZaloAPI, members any, id string) (*zago.Group, error) {
			return api.HandleGroupPending(members, id, approve)
		})
}

// MessageRef là một tin đã có, dựng lại từ những gì ĐÃ LƯU trong kho của ta
// chứ không đọc lại Zalo.
//
// Ghim, thả cảm xúc và xoá đều cần cùng bấy nhiêu trường, nên dùng chung một
// kiểu thay vì ba kiểu gần giống nhau.
type MessageRef struct {
	MsgID    string
	CliMsgID string
	OwnerID  string
	MsgType  string
	Content  string
}

func (m MessageRef) toMessageObject() *zago.MessageObject {
	return zago.NewMessageObject(map[string]any{
		"msgId": m.MsgID, "cliMsgId": m.CliMsgID, "uidFrom": m.OwnerID,
		"msgType": m.MsgType, "content": m.Content,
	})
}

// PinMessage ghim một tin trong nhóm.
func (c *Client) PinMessage(groupID string, target MessageRef) error {
	if strings.TrimSpace(target.MsgID) == "" {
		return errors.New("zalo-kit: thiếu mã tin cần ghim")
	}
	return c.thaoTacNhom(groupID, "ghim tin", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.PinMessage(target.toMessageObject(), strings.TrimSpace(groupID))
	})
}

// UnpinMessage bỏ ghim.
func (c *Client) UnpinMessage(groupID, pinID string, pinTime int64) error {
	pinID = strings.TrimSpace(pinID)
	if pinID == "" {
		return errors.New("zalo-kit: thiếu mã ghim")
	}
	return c.thaoTacNhom(groupID, "bỏ ghim tin", func(api *zago.ZaloAPI) (*zago.Group, error) {
		return api.UnpinMessage(pinID, pinTime, strings.TrimSpace(groupID))
	})
}

// SendReaction thả cảm xúc lên một tin.
//
// Dùng được cả luồng riêng lẫn nhóm; để ở đây cùng MessageRef thay vì tách
// thêm một tệp nữa.
func (c *Client) SendReaction(threadID string, threadType inbound.ThreadType,
	target MessageRef, icon string) error {
	threadID, icon = strings.TrimSpace(threadID), strings.TrimSpace(icon)
	if threadID == "" || strings.TrimSpace(target.MsgID) == "" || icon == "" {
		return errors.New("zalo-kit: thiếu mã luồng, mã tin hoặc biểu tượng")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.SendReaction(target.toMessageObject(), icon, threadID,
		zaloThreadType(threadType), 0); err != nil {
		return fmt.Errorf("zalo-kit: thả cảm xúc: %w", err)
	}
	return nil
}

// DeleteMessage xoá một tin khỏi hội thoại.
//
// ⚠️ onlyMe=true là xoá PHÍA MÌNH, người kia vẫn thấy. Đây KHÔNG phải thu hồi
// — thu hồi là UndoMessage. Lẫn hai cái là hứa với người dùng một việc rồi làm
// một việc khác.
func (c *Client) DeleteMessage(threadID string, target MessageRef, onlyMe bool) error {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" || strings.TrimSpace(target.MsgID) == "" {
		return errors.New("zalo-kit: thiếu mã luồng hoặc mã tin")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	api, err := c.guard()
	if err != nil {
		return err
	}
	if _, err = api.DeleteMessage(target.MsgID, target.OwnerID, target.CliMsgID,
		threadID, onlyMe); err != nil {
		return fmt.Errorf("zalo-kit: xoá tin: %w", err)
	}
	return nil
}

// thaoTacNhom gom khuôn chung: chặn mã rỗng, khoá, kiểm phiên, gọi, bọc lỗi.
func (c *Client) thaoTacNhom(groupID, viec string, goi func(*zago.ZaloAPI) (*zago.Group, error)) error {
	if strings.TrimSpace(groupID) == "" {
		return errors.New("zalo-kit: thiếu mã nhóm")
	}
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

func (c *Client) thaoTacThanhVien(groupID string, userIDs []string, viec string,
	goi func(*zago.ZaloAPI, any, string) (*zago.Group, error)) error {
	members := locMa(userIDs)
	if len(members) == 0 {
		return errors.New("zalo-kit: thiếu danh sách người dùng")
	}
	return c.thaoTacNhom(groupID, viec, func(api *zago.ZaloAPI) (*zago.Group, error) {
		return goi(api, members, strings.TrimSpace(groupID))
	})
}

func (c *Client) nhomTheoCay(goi func() (any, error), viec string) ([]Group, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.guard(); err != nil {
		return nil, err
	}
	raw, err := goi()
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: %s: %w", viec, err)
	}
	found := map[string]Group{}
	collectGroups(raw, found, 0)
	out := make([]Group, 0, len(found))
	for _, group := range found {
		out = append(out, group)
	}
	return out, nil
}

// locMa bỏ mã rỗng và mã trùng. Gửi mã trùng lên Zalo là hai lời mời cho cùng
// một người.
func locMa(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func collectGroups(value any, out map[string]Group, depth int) {
	if depth > 6 || value == nil {
		return
	}
	switch current := value.(type) {
	case map[string]any:
		if group, ok := groupFromMap(current); ok {
			out[group.GroupID] = group
		}
		for _, nested := range current {
			collectGroups(nested, out, depth+1)
		}
	case []any:
		for _, nested := range current {
			collectGroups(nested, out, depth+1)
		}
	case interface{ ToMap() map[string]any }:
		collectGroups(current.ToMap(), out, depth+1)
	}
}

// groupFromMap đòi ĐỦ mã nhóm và tên, cùng lý do như profileFromMap: node
// thiếu một trong hai là mảnh dữ liệu khác, ghi vào sẽ đè hỏng nhóm đang đúng.
func groupFromMap(raw map[string]any) (Group, bool) {
	id := firstID(raw, "groupId", "groupID", "grid", "group_id")
	name := firstString(raw, "name", "groupName", "displayName", "group_name")
	if id == "" || name == "" {
		return Group{}, false
	}
	count := 0
	if n, err := strconv.Atoi(firstID(raw, "totalMember", "memberCount", "total_member", "currentMems")); err == nil {
		count = n
	}
	return Group{
		GroupID:     id,
		Name:        name,
		Avatar:      firstString(raw, "avt", "avatar", "fullAvt", "avatarUrl"),
		OwnerID:     firstID(raw, "creatorId", "ownerId", "creator_id"),
		MemberCount: count,
	}, true
}

// maNhomTran nhặt mã nhóm khi Zalo trả về mà KHÔNG kèm tên — lúc vừa lập nhóm
// thì groupFromMap không nhận ra vì thiếu tên.
func maNhomTran(value any, depth int) string {
	if depth > 6 {
		return ""
	}
	switch current := value.(type) {
	case map[string]any:
		if id := firstID(current, "groupId", "groupID", "grid", "group_id"); id != "" {
			return id
		}
		for _, nested := range current {
			if id := maNhomTran(nested, depth+1); id != "" {
				return id
			}
		}
	case []any:
		for _, nested := range current {
			if id := maNhomTran(nested, depth+1); id != "" {
				return id
			}
		}
	case interface{ ToMap() map[string]any }:
		return maNhomTran(current.ToMap(), depth+1)
	}
	return ""
}

// timDuongMoi nhặt đường mời trong cây trả về của GetGroupLink.
func timDuongMoi(value any, depth int) string {
	if depth > 6 {
		return ""
	}
	switch current := value.(type) {
	case map[string]any:
		if link := firstString(current, "link", "url", "inviteLink", "invite_link"); link != "" {
			return link
		}
		for _, nested := range current {
			if link := timDuongMoi(nested, depth+1); link != "" {
				return link
			}
		}
	case []any:
		for _, nested := range current {
			if link := timDuongMoi(nested, depth+1); link != "" {
				return link
			}
		}
	case string:
		if strings.HasPrefix(strings.TrimSpace(current), "http") {
			return strings.TrimSpace(current)
		}
	}
	return ""
}
