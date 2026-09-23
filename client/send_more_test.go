package client

import (
	"errors"
	"testing"

	"github.com/thteam47/zalo-kit/inbound"
)

// Mọi hành động gửi phải qua ĐỦ ba cửa: chặn đầu vào rỗng, kiểm phiên, rồi mới
// gọi Zalo.
//
// ⚠️ Thứ tự quan trọng: chặn đầu vào rỗng PHẢI đứng trước kiểm phiên. Ngược
// lại thì một ô để trống báo "phiên hỏng, quét QR lại" — người dùng đi làm một
// việc chẳng liên quan gì.
func TestMoiHanhDongGuiChanDauVaoRongTruoc(t *testing.T) {
	c := &Client{accountID: "acc-1"} // api == nil, tức phiên hỏng
	cases := map[string]func() error{
		"tệp":         func() error { return c.SendFile("", inbound.ThreadDirect, "", "a.pdf", "pdf", 1) },
		"video":       func() error { return c.SendVideo("", inbound.ThreadDirect, "", "", "", 1, 1, 1) },
		"thoại":       func() error { return c.SendVoice("", inbound.ThreadDirect, "", 1) },
		"nhãn dán":    func() error { return c.SendSticker("", inbound.ThreadDirect, 1, 1, 1) },
		"đường dẫn":   func() error { return c.SendLink("", inbound.ThreadDirect, "", "") },
		"nhiều ảnh":   func() error { return c.SendMultiImage("", inbound.ThreadDirect, nil, "", 1, 1) },
		"trích dẫn":   func() error { return c.SendQuote("t", inbound.ThreadDirect, "", Quote{}) },
		"thu hồi":     func() error { return c.UndoMessage("", inbound.ThreadDirect, "", "") },
		"đã xem":      func() error { return c.MarkAsRead("", inbound.ThreadDirect, "", "", "") },
		"đã nhận":     func() error { return c.MarkAsDelivered("", inbound.ThreadDirect, "", "", "") },
	}
	for ten, goi := range cases {
		err := goi()
		if err == nil {
			t.Errorf("%s: đầu vào rỗng mà vẫn cho đi", ten)
			continue
		}
		if errors.Is(err, ErrSessionInvalid) {
			t.Errorf("%s: báo nhầm là lỗi phiên, trong khi lỗi là đầu vào rỗng: %v", ten, err)
		}
	}
}

// Đầu vào đủ mà phiên hỏng thì phải nói ĐÚNG là phiên hỏng.
func TestPhienHongThiBaoPhienHong(t *testing.T) {
	c := &Client{accountID: "acc-1"}
	cases := map[string]func() error{
		"tệp":       func() error { return c.SendFile("t1", inbound.ThreadDirect, "https://x/a.pdf", "a.pdf", "pdf", 10) },
		"video":     func() error { return c.SendVideo("t1", inbound.ThreadDirect, "https://x/v.mp4", "", "", 3, 1, 1) },
		"thoại":     func() error { return c.SendVoice("t1", inbound.ThreadDirect, "https://x/v.m4a", 10) },
		"nhãn dán":  func() error { return c.SendSticker("t1", inbound.ThreadDirect, 1, 2, 3) },
		"đường dẫn": func() error { return c.SendLink("t1", inbound.ThreadDirect, "https://x", "xem nhé") },
		"nhiều ảnh": func() error { return c.SendMultiImage("t1", inbound.ThreadDirect, []string{"https://x/1.jpg"}, "", 1, 1) },
		"trích dẫn": func() error { return c.SendQuote("t1", inbound.ThreadDirect, "dạ", Quote{MsgID: "m1"}) },
		"thu hồi":   func() error { return c.UndoMessage("t1", inbound.ThreadDirect, "m1", "c1") },
		"đã xem":    func() error { return c.MarkAsRead("t1", inbound.ThreadDirect, "m1", "c1", "u1") },
	}
	for ten, goi := range cases {
		if err := goi(); !errors.Is(err, ErrSessionInvalid) {
			t.Errorf("%s: phải là ErrSessionInvalid, được: %v", ten, err)
		}
	}
}
