package client

import "testing"

// Thẻ link Zalo: content là object — phải giữ đủ href / ảnh / params để hộp thư vẽ lại thẻ.
func TestCardOfGiuKhoiTheLink(t *testing.T) {
	raw := map[string]any{"msgType": "chat.recommended", "content": map[string]any{
		"action": "recommened.link", "href": "https://shope.ee/abc", "title": "https://shope.ee/abc",
		"description": "Son dưỡng", "thumb": "https://img/1.jpg", "params": `{"mediaTitle":"Son"}`,
	}}
	c := cardOf(raw)
	if c == nil || c.Action != "recommened.link" || c.Href != "https://shope.ee/abc" || c.Thumb != "https://img/1.jpg" || c.Params != `{"mediaTitle":"Son"}` || c.Description != "Son dưỡng" {
		t.Fatalf("thẻ sai: %#v", c)
	}
	v := cardOf(map[string]any{"content": map[string]any{"href": "https://v/1.mp4", "params": map[string]any{"duration": float64(12000)}}})
	if v == nil || v.Href != "https://v/1.mp4" || v.Params != `{"duration":12000}` {
		t.Fatalf("video sai: %#v", v)
	}
	if cardOf(map[string]any{"content": "chào"}) != nil || cardOf(map[string]any{"content": map[string]any{}}) != nil {
		t.Fatal("content chuỗi / rỗng thì không có thẻ")
	}
}
