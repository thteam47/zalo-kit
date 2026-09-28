package client

import (
	"testing"

	"github.com/thteam47/zalo-kit/inbound"
)

func TestStickerIDOfDocSoVaChuoi(t *testing.T) {
	cases := []struct {
		raw  map[string]any
		want int
	}{
		{map[string]any{"content": map[string]any{"id": float64(46991), "catId": float64(10)}}, 46991},
		{map[string]any{"content": map[string]any{"id": " 123 "}}, 123},
		{map[string]any{"content": "chữ thường"}, 0},
		{map[string]any{}, 0},
	}
	for _, c := range cases {
		if got := stickerIDOf(c.raw); got != c.want {
			t.Fatalf("stickerIDOf(%v) = %d, muốn %d", c.raw, got, c.want)
		}
	}
}

func TestFindStickerURLUuTienWebpVaDuyetLong(t *testing.T) {
	resp := map[string]any{"data": map[string]any{"list": []any{
		map[string]any{"stickerUrl": "https://z.vn/a.png", "stickerWebpUrl": "https://z.vn/a.webp"},
	}}}
	if got := findStickerURL(resp, 0); got != "https://z.vn/a.webp" {
		t.Fatalf("got %q", got)
	}
	if got := findStickerURL(map[string]any{"stickerUrl": "không phải link"}, 0); got != "" {
		t.Fatalf("chuỗi không phải link phải bị bỏ, được %q", got)
	}
}

func TestFillStickerMediaChiDungTinNhanDanThieuAnh(t *testing.T) {
	var c *Client
	// Tin chữ và tin nhãn dán ĐÃ có ảnh thì không đụng — không gọi Zalo.
	text := inbound.Message{Type: inbound.MessageText}
	if got := c.fillStickerMedia(text, nil); got.MediaURL != "" {
		t.Fatalf("tin chữ bị gắn ảnh")
	}
	stickerURLCache.Store(777, "https://z.vn/777.webp")
	msg := inbound.Message{Type: inbound.MessageSticker}
	raw := map[string]any{"content": map[string]any{"id": float64(777)}}
	if got := c.fillStickerMedia(msg, raw); got.MediaURL != "https://z.vn/777.webp" {
		t.Fatalf("không đọc bộ đệm: %q", got.MediaURL)
	}
}
