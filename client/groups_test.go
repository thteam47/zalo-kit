package client

import (
	"errors"
	"strings"
	"testing"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

// Đầu vào rỗng phải bị chặn TRƯỚC khi kiểm phiên.
//
// Không phải chuyện thẩm mỹ: client chưa đăng nhập luôn trả "phiên hỏng", nên
// một mã nhóm rỗng do lỗi ở tầng trên sẽ hiện ra màn hình là "phiên Zalo hỏng"
// và người vận hành đi quét lại QR — sửa nhầm chỗ, không bao giờ tìm ra.
func TestNhomChanDauVaoRongTruocKhiKiemPhien(t *testing.T) {
	c := &Client{} // c.api == nil ⇒ mọi lời gọi qua được guard đều trả ErrSessionInvalid
	cases := map[string]error{
		"ListGroups rỗng thì phải là lỗi phiên": func() error { _, err := c.GroupMembers(" "); return err }(),
		"GroupInfo không mã":                    func() error { _, err := c.GroupInfo(); return err }(),
		"CreateGroup không tên":                 func() error { _, err := c.CreateGroup(" ", "", []string{"u1"}); return err }(),
		"CreateGroup không thành viên":          func() error { _, err := c.CreateGroup("Nhóm", "", nil); return err }(),
		"AddGroupMembers không ai":              c.AddGroupMembers("g1", []string{" ", ""}),
		"RemoveGroupMembers không ai":           c.RemoveGroupMembers("g1", nil),
		"RenameGroup không tên":                 c.RenameGroup("g1", "  "),
		"SetGroupAvatar không ảnh":              c.SetGroupAvatar("g1", ""),
		"TransferGroupOwner không chủ mới":      c.TransferGroupOwner("g1", " "),
		"LeaveGroup không mã":                   c.LeaveGroup("", false),
		"JoinGroupByLink không đường":           c.JoinGroupByLink(" "),
		"MuteGroup không mã":                    c.MuteGroup("", true),
		"PinMessage không mã tin":               c.PinMessage("g1", MessageRef{}),
		"UnpinMessage không mã ghim":            c.UnpinMessage("g1", " ", 0),
		"SendReaction không biểu tượng":         c.SendReaction("t1", inbound.ThreadDirect, MessageRef{MsgID: "m1"}, " "),
		"DeleteMessage không mã tin":            c.DeleteMessage("t1", MessageRef{}, true),
		"CreatePoll một lựa chọn":               c.CreatePoll("g1", Poll{Question: "Ăn gì?", Options: []string{"Phở", "Phở"}}),
		"VotePoll không lựa chọn":               c.VotePoll("g1", 7, nil),
		"UploadFile không đường dẫn":            func() error { _, err := c.UploadFile("t1", inbound.ThreadDirect, " "); return err }(),
		"SendLocalImage không ảnh":              c.SendLocalImage("t1", inbound.ThreadDirect, "", "", 0, 0),
		"SendBankCard thiếu ngân hàng":          c.SendBankCard("t1", inbound.ThreadDirect, "123", "NGUYEN VAN A", " "),
		"SetAutoDeleteChat hạn âm":              c.SetAutoDeleteChat("t1", inbound.ThreadDirect, -1),
	}
	for ten, err := range cases {
		if err == nil {
			t.Errorf("%s: phải trả lỗi", ten)
			continue
		}
		if errors.Is(err, ErrSessionInvalid) {
			t.Errorf("%s: báo nhầm là lỗi phiên (%v)", ten, err)
		}
	}
}

// Ngược lại: đầu vào ĐỦ mà chưa có phiên thì phải nói đúng là phiên hỏng.
func TestNhomPhienHongThiBaoPhienHong(t *testing.T) {
	c := &Client{}
	cases := map[string]error{
		"ListGroups":      func() error { _, err := c.ListGroups(); return err }(),
		"GroupMembers":    func() error { _, err := c.GroupMembers("g1"); return err }(),
		"AddGroupMembers": c.AddGroupMembers("g1", []string{"u1"}),
		"LeaveGroup":      c.LeaveGroup("g1", false),
		"MuteGroup":       c.MuteGroup("g1", true),
		"LastMessages":    func() error { _, err := c.LastMessages(); return err }(),
		"AccountInfo":     func() error { _, err := c.AccountInfo(); return err }(),
		"GroupPolls":      func() error { _, err := c.GroupPolls("g1", Paging{}); return err }(),
	}
	for ten, err := range cases {
		if !errors.Is(err, ErrSessionInvalid) {
			t.Errorf("%s: muốn ErrSessionInvalid, nhận %v", ten, err)
		}
	}
}

// locMa bỏ trùng: gửi mã trùng lên Zalo là hai lời mời cho cùng một người.
func TestLocMaBoRongVaTrung(t *testing.T) {
	got := locMa([]string{" u1 ", "u1", "", "  ", "u2"})
	if len(got) != 2 || got[0] != "u1" || got[1] != "u2" {
		t.Fatalf("nhận %#v", got)
	}
}

// groupFromMap phải đòi ĐỦ mã và tên, cùng lý do như profileFromMap: node
// thiếu một trong hai là mảnh dữ liệu khác, ghi vào sẽ đè hỏng nhóm đang đúng.
func TestGroupFromMapDoiDuMaVaTen(t *testing.T) {
	if _, ok := groupFromMap(map[string]any{"groupId": "g1"}); ok {
		t.Error("thiếu tên mà vẫn nhận")
	}
	if _, ok := groupFromMap(map[string]any{"name": "Nhóm A"}); ok {
		t.Error("thiếu mã mà vẫn nhận")
	}
	group, ok := groupFromMap(map[string]any{
		"groupId": float64(123), "name": "Nhóm A", "avt": "https://a/b.jpg",
		"creatorId": "u9", "totalMember": float64(12),
	})
	if !ok || group.GroupID != "123" || group.MemberCount != 12 || group.OwnerID != "u9" {
		t.Fatalf("nhận %#v", group)
	}
}

// Zalo lồng nhóm sâu trong cây trả về, khác nhau tuỳ endpoint.
func TestCollectGroupsDuyetCaCay(t *testing.T) {
	out := map[string]Group{}
	collectGroups(map[string]any{
		"data": map[string]any{
			"gridVerMap": []any{
				map[string]any{"groupId": "g1", "name": "Nhóm 1"},
				map[string]any{"groupId": "g2", "name": "Nhóm 2"},
			},
		},
	}, out, 0)
	if len(out) != 2 {
		t.Fatalf("muốn 2 nhóm, nhận %d: %#v", len(out), out)
	}
}

// Đường mời và đường tệp nằm ở khoá khác nhau tuỳ loại; đọc sót thì bên gọi
// dán chuỗi rỗng vào tin nhắn.
func TestTimDuongTrongCayTraVe(t *testing.T) {
	link := timDuongMoi(map[string]any{"data": map[string]any{"link": "https://zalo.me/g/abc"}}, 0)
	if link != "https://zalo.me/g/abc" {
		t.Errorf("đường mời: nhận %q", link)
	}
	file := timDuongTep(map[string]any{"data": map[string]any{"normalUrl": "https://cdn/anh.jpg"}}, 0)
	if file != "https://cdn/anh.jpg" {
		t.Errorf("đường tệp: nhận %q", file)
	}
	if timDuongMoi(map[string]any{"data": map[string]any{"x": 1}}, 0) != "" {
		t.Error("không có đường thì phải trả rỗng")
	}
}

// ⚠️ evt.Event là con trỏ và zago có nhánh gửi nil. Panic ở đây giết luôn vòng
// rút channel, nên MỌI sự kiện sau đó biến mất im lặng.
func TestSuKienNhomChiuDuocEventNil(t *testing.T) {
	got := suKienNhom(zago.GroupEventEnvelope{EventType: "leave"})
	if got.Kind != EventGroup || got.GroupEventType != "leave" {
		t.Fatalf("nhận %#v", got)
	}
	if got.At.IsZero() {
		t.Error("vẫn phải có mốc thời gian")
	}
}

// Có nội dung thì phải nhặt được mã nhóm để bên gọi biết sự kiện thuộc luồng nào.
func TestSuKienNhomNhatMaNhom(t *testing.T) {
	got := suKienNhom(zago.GroupEventEnvelope{
		EventType: "remove_member",
		Event:     zago.NewEventObject(map[string]any{"groupId": "g7", "msgId": "m3"}),
	})
	if got.ThreadID != "g7" || got.MessageID != "m3" {
		t.Fatalf("nhận %#v", got)
	}
}

// Paging.Count = 0 làm Zalo trả RỖNG, trông hệt như "nhóm chưa có gì".
func TestPagingCountRongThiDungMacDinh(t *testing.T) {
	p := Paging{}.dung()
	if p.Count != 20 || p.Page != 1 {
		t.Fatalf("nhận %#v", p)
	}
	giu := Paging{Page: 3, Count: 50}.dung()
	if giu.Page != 3 || giu.Count != 50 {
		t.Fatalf("giá trị người gọi đặt bị đổi: %#v", giu)
	}
}

// Lời nhắn rỗng phải thành nil: một Message rỗng là một tin chữ TRỐNG gửi kèm.
func TestLoiNhanRongThanhNil(t *testing.T) {
	if loiNhan("  ") != nil {
		t.Error("lời nhắn rỗng phải là nil")
	}
	if got := loiNhan("còn hàng không shop"); got == nil || got.Text != "còn hàng không shop" {
		t.Errorf("nhận %#v", got)
	}
}

// Thông điệp lỗi phải nói được việc gì hỏng, không chỉ "zalo-kit: lỗi".
func TestLoiNoiRoViecGiHong(t *testing.T) {
	c := &Client{}
	err := c.PinMessage("g1", MessageRef{})
	if !strings.Contains(err.Error(), "mã tin") {
		t.Errorf("nhận %q", err)
	}
}
