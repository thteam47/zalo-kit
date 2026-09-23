package inbound

import "time"

// Message is the stable, transport-independent representation consumed by
// products using zalo-kit. Raw provider payloads must not escape the adapter.
type Message struct {
	ID          string
	AccountID   string
	ThreadID    string
	SenderID    string
	ThreadType  ThreadType
	Type        MessageType
	// RawType là msgType NGUYÊN VĂN của Zalo, ví dụ "webchat", "chat.photo",
	// "chat.undo".
	//
	// ⚠️ BẮT BUỘC giữ lại, vì Type đã gộp mất thông tin: bộ phân loại đổi MỌI
	// loại chứa chữ "chat" thành tin chữ. Nghĩa là một sự kiện THU HỒI (đến dưới
	// dạng chat.undo) sẽ đi vào lịch sử như một tin bình thường — tệ hơn cả việc
	// bỏ qua nó, vì nó THÊM một tin không có thật vào cuộc trò chuyện.
	//
	// Bên gọi dùng trường này để nhận ra các loại đặc biệt mà không phải đoán
	// trước danh sách đầy đủ — Zalo thêm loại mới bất cứ lúc nào.
	RawType     string
	Text        string
	MediaURL    string
	IsSelf      bool
	MentionsBot bool
	OccurredAt  time.Time
}

type ThreadType string

const (
	ThreadDirect ThreadType = "direct"
	ThreadGroup  ThreadType = "group"
)

type MessageType string

const (
	MessageText    MessageType = "text"
	MessageImage   MessageType = "image"
	MessageFile    MessageType = "file"
	MessageSticker MessageType = "sticker"
	MessageUnknown MessageType = "unknown"
)

func (m Message) Valid() bool {
	return m.ID != "" && m.AccountID != "" && m.ThreadID != "" && m.SenderID != "" && !m.OccurredAt.IsZero()
}
