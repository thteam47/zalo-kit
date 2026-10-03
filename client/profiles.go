package client

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Profile là mọi thứ Zalo cho biết về một người. Hộp thư cần tên để gọi khách,
// bot cần giới tính + tên gọi để xưng hô; còn lại (bio, ảnh bìa, username…) là
// để nhân viên hiểu khách. Lấy được gì giữ nấy: hỏi lại Zalo tốn hạn mức.
type Profile struct {
	UserID string
	// DisplayName là tên dùng để hiển thị — ưu tiên tên Zalo gốc của người đó.
	DisplayName string
	// ZaloName là tên người đó tự đặt; Alias là tên gợi nhớ CHÍNH nick shop đặt
	// cho họ (Zalo trả trong displayName khi hai bên là bạn). Giữ cả hai: tên
	// gợi nhớ hay mang thông tin quý ("Lan - sỉ Hà Đông").
	ZaloName string
	Alias    string
	Username string
	Avatar   string
	Cover    string
	Phone    string
	// Bio là dòng trạng thái/giới thiệu ("status" trên dây).
	Bio string
	// Gender là "male"/"female" nếu Zalo có trả. Rỗng là chuyện thường: khách
	// để riêng tư thì không có field này. Bên gọi PHẢI chịu được rỗng.
	Gender string
	// DOB dạng "2006-01-02" khi đọc được ĐỦ ngày-tháng-năm.
	DOB string
	// BirthdayMonthDay dạng "01-02" (tháng-ngày) — khách ẩn năm sinh (rất
	// phổ biến) vẫn chúc sinh nhật được.
	BirthdayMonthDay string
	GlobalID         string
	// Con trỏ: nil = Zalo không nói, khác với false.
	IsFriend  *bool
	IsBlocked *bool
	IsActive  *bool
	// Raw là các trường vô hướng (chuỗi/số/bool) đúng như Zalo trả — để lưu lại
	// trường mình chưa biết dùng vào việc gì mà không phải hỏi Zalo lần nữa.
	Raw map[string]any
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
	// ⚠️ Nhận CẢ hai lối viết khoá.
	//
	// FetchUserInfo trả camelCase, nhưng tra-số-điện-thoại đi một endpoint KHÁC
	// (friend/profile/get) và ta chưa thấy dạng thật của nó. Nhận thêm snake_case
	// không tốn gì; đoán thiếu thì chức năng tìm người ÂM THẦM không bao giờ ra
	// kết quả, và triệu chứng giống hệt "số này chưa có Zalo".
	uid := firstID(raw, "userId", "uid", "userid", "id", "user_id")
	name := firstString(raw, "zaloName", "displayName", "dName", "name", "username",
		"zalo_name", "display_name")
	if uid == "" || name == "" {
		return Profile{}, false
	}
	zaloName := firstString(raw, "zaloName", "zalo_name")
	alias := firstString(raw, "displayName", "display_name", "dName")
	if alias == zaloName {
		alias = ""
	}
	sdob, _ := firstScalar(raw, "sdob", "birthday", "birthDate")
	dobNum, _ := firstScalar(raw, "dob")
	dob := normalizeDOB(sdob)
	if dob == "" {
		dob = normalizeDOB(dobNum)
	}
	monthDay := ""
	if dob != "" {
		monthDay = dob[5:]
	} else {
		monthDay = normalizeMonthDay(sdob)
	}
	// ⚠️ Giới tính là SỐ trên dây (json.Number từ za-go) và 0 = NAM. firstString
	// chỉ nhận chuỗi nên từng làm rơi MỌI giới tính; cleanID lại coi "0" là rỗng
	// nên làm rơi mọi người nam. firstScalar giữ nguyên "0".
	genderRaw, _ := firstScalar(raw, "gender", "sex", "genderId")
	return Profile{
		UserID:           uid,
		DisplayName:      name,
		ZaloName:         zaloName,
		Alias:            alias,
		Username:         firstString(raw, "username", "uname", "zaloUsername"),
		Avatar:           firstString(raw, "avatar", "avatarUrl", "avatar_url", "avt"),
		Cover:            firstString(raw, "cover", "coverUrl"),
		Phone:            firstString(raw, "phoneNumber", "phone", "phone_number"),
		Bio:              firstString(raw, "status", "bio", "description"),
		Gender:           normalizeGender(genderRaw),
		DOB:              dob,
		BirthdayMonthDay: monthDay,
		GlobalID:         firstString(raw, "globalId", "global_id"),
		IsFriend:         firstFlag(raw, "isFr", "is_fr", "isFriend"),
		IsBlocked:        firstFlag(raw, "isBlocked", "is_blocked"),
		IsActive:         firstFlag(raw, "isActive", "is_active"),
		Raw:              scalarFields(raw),
	}, true
}

// firstScalar đọc giá trị vô hướng đầu tiên thành chuỗi, GIỮ "0". za-go trả số
// dưới dạng json.Number nên fmt.Sprint là cách đọc chung cho mọi kiểu số.
func firstScalar(raw map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		v, ok := raw[key]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case map[string]any, []any:
			continue
		case float64:
			return strconv.FormatFloat(t, 'f', -1, 64), true
		case string:
			if strings.TrimSpace(t) == "" {
				continue
			}
			return strings.TrimSpace(t), true
		default:
			s := strings.TrimSpace(fmt.Sprint(t))
			if s == "" || s == "<nil>" {
				continue
			}
			return s, true
		}
	}
	return "", false
}

// firstFlag đọc cờ 0/1/true/false; không có trường thì nil (Zalo không nói).
func firstFlag(raw map[string]any, keys ...string) *bool {
	value, ok := firstScalar(raw, keys...)
	if !ok {
		return nil
	}
	var flag bool
	switch strings.ToLower(value) {
	case "1", "true":
		flag = true
	case "0", "false":
		flag = false
	default:
		return nil
	}
	return &flag
}

// scalarFields chép các trường vô hướng của node hồ sơ (bỏ map/mảng lồng: đó là
// dữ liệu khác, và có thể rất to).
func scalarFields(raw map[string]any) map[string]any {
	out := make(map[string]any, len(raw))
	for key, v := range raw {
		switch t := v.(type) {
		case nil, map[string]any, []any:
			continue
		case string, bool, float64:
			out[key] = t
		default:
			out[key] = fmt.Sprint(t)
		}
	}
	return out
}

// normalizeGender đổi mã giới tính của Zalo về "male"/"female".
//
// Hồ sơ Zalo (getprofiles): 0 = Nam, 1 = Nữ. Không có giá trị nào nghĩa là
// "chưa rõ" — nên khi trường VẮNG thì bên gọi nhận rỗng, còn "0" là nam thật.
// Nhận không ra thì trả RỖNG chứ đừng đoán: đoán sai còn tệ hơn không biết.
func normalizeGender(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "male", "m", "nam", "0":
		return "male"
	case "female", "f", "nu", "nữ", "1":
		return "female"
	}
	return ""
}

// normalizeMonthDay đọc ngày sinh KHÔNG có năm ("20/05", "20-05") thành "05-20".
func normalizeMonthDay(raw string) string {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{"02/01", "02-01", "2/1", "2-1"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.Format("01-02")
		}
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
