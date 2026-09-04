package client

import "testing"

// Zalo mã hoá giới tính bằng SỐ và không thống nhất giữa các endpoint.
// Nhận không ra thì PHẢI trả rỗng: bên gọi tin tưởng giá trị này để gọi khách,
// đoán bừa là gọi sai giới nửa số khách.
func TestNormalizeGender(t *testing.T) {
	cases := map[string]string{
		"0": "male", "male": "male", "Nam": "male", "M": "male",
		"1": "female", "female": "female", "Nữ": "female", "f": "female",
		"": "", "2": "", "unknown": "", "  ": "",
	}
	for raw, want := range cases {
		if got := normalizeGender(raw); got != want {
			t.Errorf("normalizeGender(%q) = %q, muốn %q", raw, got, want)
		}
	}
}

func TestNormalizeDOB(t *testing.T) {
	cases := map[string]string{
		"1990-05-20": "1990-05-20",
		"20/05/1990": "1990-05-20",
		"1990/05/20": "1990-05-20",
		"20-05-1990": "1990-05-20",
		"643161600":  "1990-05-20", // dấu thời gian giây
		"":           "",
		"0":          "",
		"linh tinh":  "",
	}
	for raw, want := range cases {
		if got := normalizeDOB(raw); got != want {
			t.Errorf("normalizeDOB(%q) = %q, muốn %q", raw, got, want)
		}
	}
}

// Số nhỏ lọt vào ô ngày sinh sẽ thành năm 1970 và biến MỌI khách thành "cô/chú".
func TestNormalizeDOBChanSoVoLy(t *testing.T) {
	for _, raw := range []string{"1", "12345", "99999999999999"} {
		if got := normalizeDOB(raw); got != "" {
			t.Errorf("normalizeDOB(%q) phải rỗng, được %q", raw, got)
		}
	}
}

// Hồ sơ không có giới tính/ngày sinh vẫn phải đọc được tên — đó là ca THƯỜNG
// GẶP nhất, khách để riêng tư thì Zalo không trả hai field kia.
func TestProfileFromMapThieuGioiTinhVanLayDuocTen(t *testing.T) {
	got, ok := profileFromMap(map[string]any{"userId": "u1", "displayName": "Trần Thị Lan"})
	if !ok || got.DisplayName != "Trần Thị Lan" {
		t.Fatalf("phải đọc được tên, được %+v ok=%v", got, ok)
	}
	if got.Gender != "" || got.DOB != "" {
		t.Errorf("thiếu thì phải rỗng, được gender=%q dob=%q", got.Gender, got.DOB)
	}
}

func TestProfileFromMapLayGioiTinhVaNgaySinh(t *testing.T) {
	got, ok := profileFromMap(map[string]any{
		"userId": "u2", "zaloName": "Nguyễn Văn Minh", "gender": "0", "sdob": "1990-05-20",
	})
	if !ok || got.Gender != "male" || got.DOB != "1990-05-20" {
		t.Errorf("được %+v", got)
	}
}

// Zalo có endpoint trả mili giây. Không chia 1000 thì ra năm 50000 rồi bị
// loại, mất luôn ngày sinh đọc được.
func TestNormalizeDOBNhanMiliGiay(t *testing.T) {
	if got := normalizeDOB("643161600000"); got != "1990-05-20" {
		t.Errorf("mili giây phải ra 1990-05-20, được %q", got)
	}
}
