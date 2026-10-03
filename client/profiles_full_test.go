package client

import (
	"encoding/json"
	"testing"
)

// Dữ liệu THẬT từ za-go: mọi số là json.Number (dec.UseNumber). Bản cũ đọc giới
// tính bằng firstString nên json.Number rơi mất — ZChốt không bao giờ biết giới
// tính khách. Và 0 trên dây là NAM, không phải "chưa rõ".
func TestProfileFromMapGioiTinhSoJSONNumber(t *testing.T) {
	cases := []struct {
		raw  any
		want string
	}{
		{json.Number("0"), "male"},
		{json.Number("1"), "female"},
		{float64(0), "male"},
		{float64(1), "female"},
		{"0", "male"},
	}
	for _, c := range cases {
		got, ok := profileFromMap(map[string]any{"userId": "u1", "zaloName": "Minh", "gender": c.raw})
		if !ok || got.Gender != c.want {
			t.Errorf("gender=%#v → %q, muốn %q", c.raw, got.Gender, c.want)
		}
	}
	got, _ := profileFromMap(map[string]any{"userId": "u1", "zaloName": "Minh"})
	if got.Gender != "" {
		t.Errorf("thiếu trường gender phải rỗng, được %q", got.Gender)
	}
}

func TestProfileFromMapLayDuHoSo(t *testing.T) {
	got, ok := profileFromMap(map[string]any{
		"userId": json.Number("123456789"), "zaloName": "Trần Thị Lan", "displayName": "Lan sỉ Hà Đông",
		"username": "lantran", "avatar": "https://a/x.jpg", "cover": "https://a/c.jpg",
		"status": "Chuyên sỉ quần áo trẻ em", "gender": json.Number("1"),
		"sdob": "20/05", "globalId": "G1", "isFr": json.Number("1"), "isBlocked": json.Number("0"),
		"bizPkg": map[string]any{"label": nil},
	})
	if !ok {
		t.Fatal("phải đọc được hồ sơ")
	}
	if got.ZaloName != "Trần Thị Lan" || got.Alias != "Lan sỉ Hà Đông" || got.Username != "lantran" {
		t.Errorf("tên sai: %+v", got)
	}
	if got.Bio != "Chuyên sỉ quần áo trẻ em" || got.Cover == "" || got.GlobalID != "G1" {
		t.Errorf("thiếu bio/cover/globalId: %+v", got)
	}
	if got.Gender != "female" || got.DOB != "" || got.BirthdayMonthDay != "05-20" {
		t.Errorf("giới tính/ngày sinh sai: gender=%q dob=%q md=%q", got.Gender, got.DOB, got.BirthdayMonthDay)
	}
	if got.IsFriend == nil || !*got.IsFriend || got.IsBlocked == nil || *got.IsBlocked || got.IsActive != nil {
		t.Errorf("cờ sai: fr=%v blocked=%v active=%v", got.IsFriend, got.IsBlocked, got.IsActive)
	}
	if got.Raw["status"] != "Chuyên sỉ quần áo trẻ em" || got.Raw["gender"] != "1" {
		t.Errorf("Raw phải giữ trường vô hướng: %#v", got.Raw)
	}
	if _, has := got.Raw["bizPkg"]; has {
		t.Error("Raw không được chép map lồng")
	}
}

func TestProfileFromMapNgaySinhDuNam(t *testing.T) {
	got, _ := profileFromMap(map[string]any{"userId": "u1", "zaloName": "A", "sdob": "20/05/1990"})
	if got.DOB != "1990-05-20" || got.BirthdayMonthDay != "05-20" {
		t.Errorf("dob=%q md=%q", got.DOB, got.BirthdayMonthDay)
	}
	got, _ = profileFromMap(map[string]any{"userId": "u1", "zaloName": "A", "dob": json.Number("643161600")})
	if got.DOB != "1990-05-20" {
		t.Errorf("dob số giây: %q", got.DOB)
	}
}
