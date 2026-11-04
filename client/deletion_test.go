package client

import (
	"testing"
	"time"

	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
)

// Sự cố thật (11/2026): chủ nick xoá một tin trên điện thoại, socket dội về
// msgType "chat.delete" với content là MẢNG một phần tử. Hộp thư hiện nguyên
// khối "[map[actionType:0 clientDelMsgId:1793816118202 …]]" như một tin chữ.
func TestXoaTinPhiaZaloThanhSuKienDeletion(t *testing.T) {
	data := zago.NewMessageObject(map[string]any{
		"msgId": "m-del", "uidFrom": "me", "msgType": "chat.delete", "ts": float64(1_700_000_000_000),
		"content": []any{map[string]any{
			"actionType": float64(0), "clientDelMsgId": float64(1793816118202), "destId": float64(8375422841292060205),
			"globalDelMsgId": float64(0), "type": float64(2), "uidFrom": float64(3391991), "uidTo": float64(8375422841292060205),
		}},
	})
	got := normalizeMessage("acc", "me", "", "me", "[map[actionType:0 clientDelMsgId:1793816118202 destId:8375422841292060205 globalDelMsgId:0 type:2]]",
		data, "8375422841292060205", zago.ThreadTypeUSER, time.Time{})
	if got.Type != inbound.MessageDeletion || got.Deletion == nil {
		t.Fatalf("phải là sự kiện xoá, nhận %#v", got)
	}
	if got.Deletion.Kind != inbound.DeletionDelete {
		t.Fatalf("chat.delete là xoá phía người bấm, nhận %q", got.Deletion.Kind)
	}
	if got.Deletion.TargetClientMessageID != "1793816118202" || got.Deletion.TargetMessageID != "" {
		t.Fatalf("phải trỏ đúng cliMsgId tin gốc (globalDelMsgId 0 = chưa biết), nhận %#v", got.Deletion)
	}
	if got.Text != "" {
		t.Fatalf("khối map không được thành chữ, nhận %q", got.Text)
	}
	if !got.IsSelf {
		t.Fatal("uidFrom là chính nick thì phải IsSelf")
	}
}

// Thu hồi (chat.undo) để mã tin gốc ngay trong content (object).
func TestThuHoiThanhSuKienDeletionUndo(t *testing.T) {
	data := zago.NewMessageObject(map[string]any{
		"msgId": "m-undo", "uidFrom": "u-9", "msgType": "chat.undo", "ts": float64(1_700_000_000_000),
		"content": map[string]any{"msgId": "555", "cliMsgId": "444", "deleteMsg": true},
	})
	got := normalizeMessage("acc", "me", "", "u-9", "", data, "u-9", zago.ThreadTypeUSER, time.Time{})
	if got.Type != inbound.MessageDeletion || got.Deletion == nil || got.Deletion.Kind != inbound.DeletionUndo {
		t.Fatalf("phải là thu hồi, nhận %#v", got)
	}
	if got.Deletion.TargetMessageID != "555" || got.Deletion.TargetClientMessageID != "444" {
		t.Fatalf("phải trỏ đúng tin gốc, nhận %#v", got.Deletion)
	}
	if got.RawType != "chat.undo" {
		t.Fatalf("giữ nguyên msgType gốc, nhận %q", got.RawType)
	}
}

// Content là chuỗi JSON cũng phải đọc được; content trống thì vẫn là sự kiện
// xoá (bên nhận tự xử khi không biết tin đích).
func TestDeletionContentChuoiJsonHoacTrong(t *testing.T) {
	data := zago.NewMessageObject(map[string]any{
		"msgId": "m-1", "uidFrom": "u-9", "msgType": "group.undo",
		"content": `{"msgId":"777","cliMsgId":"666"}`,
	})
	got := normalizeMessage("acc", "me", "", "u-9", "", data, "g-1", zago.ThreadTypeGROUP, time.Time{})
	if got.Deletion == nil || got.Deletion.TargetMessageID != "777" || got.Deletion.TargetClientMessageID != "666" {
		t.Fatalf("content JSON phải đọc được, nhận %#v", got.Deletion)
	}
	empty := normalizeMessage("acc", "me", "", "u-9", "", zago.NewMessageObject(map[string]any{"msgId": "m-2", "uidFrom": "u-9", "msgType": "chat.delete"}),
		"u-9", zago.ThreadTypeUSER, time.Time{})
	if empty.Type != inbound.MessageDeletion || empty.Deletion == nil || empty.Deletion.TargetClientMessageID != "" {
		t.Fatalf("content trống vẫn là sự kiện xoá, nhận %#v", empty)
	}
}

// Tin thường tuyệt đối không bị nhận nhầm thành xoá — nhầm là MẤT tin.
func TestTinThuongKhongThanhDeletion(t *testing.T) {
	for _, raw := range []string{"", "webchat", "chat.photo", "chat.recommended", "chat.voice", "chat.sticker"} {
		got := normalizeMessage("acc", "me", "m1", "u-9", "", zago.NewMessageObject(map[string]any{"msgId": "m1", "uidFrom": "u-9", "msgType": raw, "content": "alo"}),
			"u-9", zago.ThreadTypeUSER, time.Time{})
		if got.Deletion != nil || got.Type == inbound.MessageDeletion {
			t.Errorf("msgType %q KHÔNG phải xoá: %#v", raw, got)
		}
	}
}

// Khối "[map[…]]" (mảng map in ra) cũng là rác, không được lưu làm chữ.
func TestDumpMangMapCungKhongLuu(t *testing.T) {
	if !looksLikeDump("[map[actionType:0 clientDelMsgId:1]]") || !looksLikeDump("map[a:1]") || looksLikeDump("[1] mở hàng") {
		t.Fatal("looksLikeDump phải bắt cả map[ lẫn [map[, nhưng không bắt chữ thường")
	}
}
