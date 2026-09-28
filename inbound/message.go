package inbound

import "time"

// Message is the stable, transport-independent representation consumed by
// products using zalo-kit. Raw provider payloads must not escape the adapter.
type Message struct {
	ID         string
	AccountID  string
	ThreadID   string
	SenderID   string
	ThreadType ThreadType
	Type       MessageType
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
	// ThreadGlobalID là globalId của luồng (của khách, với chat riêng).
	//
	// zalo-kit KHÔNG tự điền: đổi uid ra globalId là một lời gọi Zalo, và bên
	// nhận tin mới biết lúc nào nên gọi, lúc nào đọc lại từ bộ đệm của mình.
	// Trường nằm ở đây để giá trị đó đi cùng tin qua mọi tầng phía sau.
	ThreadGlobalID string
	// ClientMessageID là cliMsgId của Zalo. Muốn TRẢ LỜI (trích dẫn) một tin
	// thì Zalo đòi cả msgId lẫn cliMsgId của tin đó — thiếu là không trích được.
	ClientMessageID string
	// Quote có mặt khi tin này trả lời (trích dẫn) một tin khác.
	Quote *Quote
}

// Quote là một tin được trích dẫn: dùng cả khi ĐỌC (tin đến trả lời tin nào)
// lẫn khi GỬI (mình trả lời tin nào).
type Quote struct {
	// OwnerID là uid người viết tin gốc. Để rỗng khi gửi = tin gốc của chính
	// nick đang gửi; zalo-kit tự điền uid của nick.
	OwnerID         string
	MessageID       string // msgId / globalMsgId của tin gốc
	ClientMessageID string // cliMsgId của tin gốc
	MsgType         string // msgType gốc, vd "webchat"; rỗng = "webchat"
	Text            string // phần chữ của tin gốc
	// OccurredAt là lúc tin gốc được gửi; Zalo cần mốc này để dựng khối trích.
	OccurredAt time.Time
}

// CanSend: đủ thông tin để Zalo dựng khối trích dẫn.
func (q Quote) CanSend() bool {
	return q.MessageID != "" && q.ClientMessageID != ""
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
