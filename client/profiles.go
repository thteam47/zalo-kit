package client

import (
	"fmt"
	"strings"
)

// Profile là tên và ảnh đại diện của một người trên Zalo. Hộp thư cần nó để gọi
// khách bằng tên thật thay vì dãy số thread id.
type Profile struct {
	UserID      string
	DisplayName string
	Avatar      string
	Phone       string
}

// FetchProfiles đọc thông tin của nhiều uid trong một lần gọi.
//
// Zalo trả về nhiều dạng lồng nhau tuỳ phiên bản endpoint (khoá "<uid>_0",
// "changed_profiles", hoặc mảng). Thay vì đoán một dạng rồi vỡ khi Zalo đổi,
// hàm này duyệt cả cây và nhặt mọi node trông như một hồ sơ người dùng.
func (c *Client) FetchProfiles(userIDs ...string) (map[string]Profile, error) {
	ids := make([]string, 0, len(userIDs))
	seen := map[string]bool{}
	for _, id := range userIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return map[string]Profile{}, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.api == nil {
		return nil, ErrSessionInvalid
	}
	raw, err := c.api.FetchUserInfo(ids...)
	if err != nil {
		return nil, fmt.Errorf("zalo-kit: read Zalo contact info: %w", err)
	}
	out := map[string]Profile{}
	collectProfiles(raw, out, 0)
	return out, nil
}

func collectProfiles(value any, out map[string]Profile, depth int) {
	if depth > 6 || value == nil {
		return
	}
	switch current := value.(type) {
	case map[string]any:
		if profile, ok := profileFromMap(current); ok {
			out[profile.UserID] = profile
		}
		for _, nested := range current {
			collectProfiles(nested, out, depth+1)
		}
	case []any:
		for _, nested := range current {
			collectProfiles(nested, out, depth+1)
		}
	case map[string]string:
		plain := make(map[string]any, len(current))
		for key, nested := range current {
			plain[key] = nested
		}
		collectProfiles(plain, out, depth+1)
	case interface{ ToMap() map[string]any }:
		collectProfiles(current.ToMap(), out, depth+1)
	}
}

// profileFromMap chỉ nhận node có ĐỦ uid và tên: thiếu một trong hai thì đó là
// mảnh dữ liệu khác, ghi vào sẽ đè hỏng hồ sơ đang đúng.
func profileFromMap(raw map[string]any) (Profile, bool) {
	uid := firstID(raw, "userId", "uid", "userid", "id")
	name := firstString(raw, "zaloName", "displayName", "dName", "name", "username")
	if uid == "" || name == "" {
		return Profile{}, false
	}
	return Profile{
		UserID:      uid,
		DisplayName: name,
		Avatar:      firstString(raw, "avatar", "avatarUrl", "avatar_url"),
		Phone:       firstString(raw, "phoneNumber", "phone"),
	}, true
}
