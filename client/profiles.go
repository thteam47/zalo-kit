package client

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Profile là tên và ảnh đại diện của một người trên Zalo. Hộp thư cần nó để gọi
// khách bằng tên thật thay vì dãy số thread id.
type Profile struct {
	UserID      string
	DisplayName string
	Avatar      string
	Phone       string
	// Gender là "male"/"female" nếu Zalo có trả. Zalo mã hoá bằng SỐ và không
	// thống nhất giữa các endpoint, nên đọc rộng rồi chuẩn hoá ở normalizeGender.
	//
	// Rỗng là chuyện thường: khách để riêng tư thì không có field này. Bên gọi
	// PHẢI chịu được rỗng chứ đừng coi là lỗi.
	Gender string
	// DOB dạng "2006-01-02" nếu đọc được.
	DOB string
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
		Gender:      normalizeGender(firstString(raw, "gender", "sex", "genderId")),
		DOB:         normalizeDOB(firstString(raw, "sdob", "dob", "birthday", "birthDate")),
	}, true
}

// normalizeGender đổi mã giới tính của Zalo về "male"/"female".
//
// Zalo trả SỐ và không thống nhất: chỗ 0/1, chỗ 1/2, chỗ lại là chữ. Nhận
// không ra thì trả RỖNG chứ đừng đoán — đoán sai còn tệ hơn không biết, vì
// bên gọi sẽ tin tưởng gọi khách sai giới.
func normalizeGender(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "male", "m", "nam", "0":
		return "male"
	case "female", "f", "nu", "nữ", "1":
		return "female"
	}
	return ""
}

// normalizeDOB đưa ngày sinh về dạng 2006-01-02.
//
// Zalo trả nhiều kiểu tuỳ endpoint: "1990-05-20", "20/05/1990", hoặc dấu thời
// gian giây. Không đọc được thì trả rỗng — thiếu ngày sinh chỉ mất tính năng
// gọi cô/chú, còn đọc nhầm năm thì gọi một người 30 tuổi là "cô".
func normalizeDOB(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return ""
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006", "2006/01/02", "02-01-2006"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	// Dấu thời gian. Zalo có endpoint trả GIÂY, có endpoint trả MILI GIÂY.
	number, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return ""
	}
	// Ngưỡng 1e10: mốc giây lớn nhất còn hợp lý (năm 2286) vẫn dưới nó, còn
	// mốc mili giây nhỏ nhất còn hợp lý (năm 1973) đã trên nó — tách sạch.
	if number >= 1e10 || number <= -1e10 {
		number /= 1000
	}
	// Chặn số nhỏ: "1" hay "12345" lọt vào đây sẽ ra 1970-01-01, và thế là MỌI
	// khách thành 56 tuổi rồi bị gọi "cô/chú". Mốc 1e8 ứng với năm 1973 — hy
	// sinh vài người sinh 1970-1972 còn hơn gọi nhầm cả tệp khách.
	if number > 0 && number < 1e8 {
		return ""
	}
	at := time.Unix(number, 0).UTC()
	if at.Year() < 1900 || at.Year() > time.Now().Year() {
		return ""
	}
	return at.Format("2006-01-02")
}
