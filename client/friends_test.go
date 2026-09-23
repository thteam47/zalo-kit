package client

import (
	"errors"
	"strings"
	"testing"
)

// Không có phiên thì báo ĐÚNG lý do, đừng để rơi vào nhánh "không tìm thấy".
//
// Hai chuyện này dẫn tới hai việc khác nhau của người dùng: "số này chưa có
// Zalo, nhập số khác" so với "nick mất phiên, quét QR lại". Gộp là họ đi sửa
// nhầm chỗ.
func TestFindByPhoneKhongCoPhienThiBaoPhienHong(t *testing.T) {
	c := &Client{accountID: "acc-1"}
	if _, err := c.FindByPhone("0901234567"); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("phải là ErrSessionInvalid, được: %v", err)
	}
}

func TestAddFriendKhongCoPhienThiBaoPhienHong(t *testing.T) {
	c := &Client{accountID: "acc-1"}
	if err := c.AddFriend("uid-1", "chào bạn"); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("phải là ErrSessionInvalid, được: %v", err)
	}
}

// Chặn từ đầu, đừng gọi Zalo với chuỗi rỗng rồi nhận một lỗi khó hiểu.
func TestChanDauVaoRong(t *testing.T) {
	c := &Client{accountID: "acc-1"}
	if _, err := c.FindByPhone("   "); err == nil || errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("số rỗng phải bị chặn trước khi kiểm phiên, được: %v", err)
	}
	if err := c.AddFriend("  ", "x"); err == nil || errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("uid rỗng phải bị chặn trước khi kiểm phiên, được: %v", err)
	}
}

// ⚠️ Tra-số đi endpoint KHÁC (friend/profile/get) so với FetchUserInfo, và ta
// chưa thấy dạng thật của nó. Bộ nhặt hồ sơ phải đọc được CẢ HAI lối viết
// khoá — đoán thiếu thì chức năng tìm người ÂM THẦM không bao giờ ra kết quả,
// mà triệu chứng lại giống hệt "số này chưa có Zalo".
func TestNhatHoSoTuDangZaloTraVe(t *testing.T) {
	for ten, node := range map[string]map[string]any{
		"camelCase":  {"uid": "123456789", "displayName": "Chị Lan", "avatar": "https://x/y.jpg"},
		"snake_case": {"user_id": "123456789", "display_name": "Chị Lan"},
	} {
		found := map[string]Profile{}
		collectProfiles(map[string]any{"data": node}, found, 0)
		if len(found) != 1 {
			t.Fatalf("%s: phải nhặt được đúng một hồ sơ, được %d", ten, len(found))
		}
		if found["123456789"].DisplayName != "Chị Lan" {
			t.Fatalf("%s: đọc sai tên: %+v", ten, found["123456789"])
		}
	}
}

// Câu lỗi phải nói được cho người dùng đọc, không phải chỉ mã lỗi.
func TestCauLoiKhongTimThayDocDuoc(t *testing.T) {
	if !strings.Contains(ErrContactNotFound.Error(), "không tìm thấy") {
		t.Fatalf("câu lỗi khó hiểu: %v", ErrContactNotFound)
	}
}
