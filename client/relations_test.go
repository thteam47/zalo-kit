package client

import (
	"errors"
	"testing"
)

// Đầu vào rỗng phải bị chặn TRƯỚC khi kiểm phiên — cùng lý do với nhóm gửi:
// ô để trống mà báo "phiên hỏng" là đẩy người dùng đi quét QR vô ích.
func TestQuanHeChanDauVaoRongTruoc(t *testing.T) {
	c := &Client{accountID: "acc-1"}
	cases := map[string]func() error{
		"đồng ý kết bạn": func() error { return c.AcceptFriendRequest(" ") },
		"huỷ kết bạn":    func() error { return c.UnfriendUser("") },
		"chặn":           func() error { return c.BlockUser("") },
		"bỏ chặn":        func() error { return c.UnblockUser("") },
		"tên gợi nhớ":    func() error { return c.SetAlias("", "Chị Lan") },
	}
	for ten, goi := range cases {
		err := goi()
		if err == nil || errors.Is(err, ErrSessionInvalid) {
			t.Errorf("%s: phải chặn đầu vào rỗng trước, được: %v", ten, err)
		}
	}
}

func TestQuanHePhienHongThiBaoDung(t *testing.T) {
	c := &Client{accountID: "acc-1"}
	cases := map[string]func() error{
		"đồng ý kết bạn": func() error { return c.AcceptFriendRequest("u1") },
		"huỷ kết bạn":    func() error { return c.UnfriendUser("u1") },
		"chặn":           func() error { return c.BlockUser("u1") },
		"tên gợi nhớ":    func() error { return c.SetAlias("u1", "Chị Lan") },
	}
	for ten, goi := range cases {
		if err := goi(); !errors.Is(err, ErrSessionInvalid) {
			t.Errorf("%s: phải là ErrSessionInvalid, được: %v", ten, err)
		}
	}
	for ten, goi := range map[string]func() ([]Profile, error){
		"danh bạ":        c.ListFriends,
		"lời mời đến":    c.ListFriendRequests,
		"lời mời đã gửi": c.ListSentFriendRequests,
	} {
		if _, err := goi(); !errors.Is(err, ErrSessionInvalid) {
			t.Errorf("%s: phải là ErrSessionInvalid, được: %v", ten, err)
		}
	}
}

// Danh sách phải trả []Profile đã nhặt sẵn, không đẩy cây JSON thô ra ngoài.
// Để `any` chạy ra là mỗi nơi gọi tự đoán một dạng, nơi nào đoán sai thì im
// lặng trả rỗng.
func TestDanhSachTraVeHoSoDaNhat(t *testing.T) {
	found := map[string]Profile{}
	collectProfiles(map[string]any{"data": []any{
		map[string]any{"userId": "u1", "displayName": "Chị Lan"},
		map[string]any{"userId": "u2", "zaloName": "Anh Nam"},
	}}, found, 0)
	if len(found) != 2 {
		t.Fatalf("phải nhặt được 2 hồ sơ, được %d", len(found))
	}
}
