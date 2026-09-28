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
		{map[string]any{"content": `{"id":46991,"catId":10,"type":7}`}, 46991},
		{map[string]any{}, 0},
	}
	for _, c := range cases {
		if got := stickerIDOf(c.raw); got != c.want {
			t.Fatalf("stickerIDOf(%v) = %d, muốn %d", c.raw, got, c.want)
		}
	}
}

func TestFillStickerMediaDungLinkTinhTheoMa(t *testing.T) {
	var c *Client
	// Tin chữ không bị gắn ảnh.
	if got := c.fillStickerMedia(inbound.Message{Type: inbound.MessageText}, nil); got.MediaURL != "" {
		t.Fatalf("tin chữ bị gắn ảnh")
	}
	msg := inbound.Message{Type: inbound.MessageSticker}
	raw := map[string]any{"content": map[string]any{"id": float64(46991), "catId": float64(10)}}
	want := "https://zalo-api.zadn.vn/api/emoticon/sticker/webpc?eid=46991&size=130"
	if got := c.fillStickerMedia(msg, raw); got.MediaURL != want {
		t.Fatalf("got %q", got.MediaURL)
	}
	// Không đọc được mã thì để trống, không bịa link.
	if got := c.fillStickerMedia(msg, map[string]any{}); got.MediaURL != "" {
		t.Fatalf("thiếu mã mà vẫn có link: %q", got.MediaURL)
	}
}
