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
	// Reaction có mặt khi đây là SỰ KIỆN thả cảm xúc lên một tin (Type =
	// MessageReaction) — KHÔNG phải một tin mới. Bên nhận phải gắn nó vào tin
	// đích, đừng ghi thành một dòng trong hội thoại.
	Reaction *Reaction
	// Call có mặt khi đây là bản ghi một cuộc gọi Zalo (Type = MessageCall).
	Call *Call
	// Deletion có mặt khi đây là SỰ KIỆN rút lại / xoá một tin đã gửi (Type =
	// MessageDeletion) — KHÔNG phải một tin mới. Bên nhận phải đánh dấu lên tin
	// đích (giữ nguyên lịch sử), đừng ghi thành một dòng trong hội thoại.
	Deletion *Deletion
}

// Deletion mô tả tin nào bị rút lại hoặc xoá.
//
// Hình dạng thật soi trên phiên sống (11/2026): msgType "chat.delete", content
// là MỘT PHẦN TỬ trong mảng: {actionType:0, clientDelMsgId:<cliMsgId tin gốc>,
// globalDelMsgId:0|<msgId>, destId, type, uidFrom, uidTo}. Thu hồi (chat.undo)
// để msgId/cliMsgId của tin gốc trong content.
type Deletion struct {
	// Kind: "undo" = thu hồi (mất ở cả hai phía Zalo) | "delete" = xoá phía
	// người bấm (phía kia vẫn thấy).
	Kind string
	// TargetMessageID / TargetClientMessageID: msgId / cliMsgId của tin bị xoá.
	TargetMessageID       string
	TargetClientMessageID string
}

const (
	DeletionUndo   = "undo"
	DeletionDelete = "delete"
)

// Reaction là một lượt thả (hoặc gỡ, khi Icon rỗng) cảm xúc lên một tin.
type Reaction struct {
	// Icon là mã Zalo: "/-heart", "/-strong", ":>"… Rỗng = gỡ cảm xúc.
	Icon string
	// TargetMessageID / TargetClientMessageID: msgId / cliMsgId của tin đích.
	TargetMessageID       string
	TargetClientMessageID string
}

// Call là bản ghi một cuộc gọi thoại/video trong hội thoại.
type Call struct {
	Video       bool
	Missed      bool
	DurationSec int
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
	// MessageReaction: sự kiện thả cảm xúc, xem Message.Reaction.
	MessageReaction MessageType = "reaction"
	// MessageCall: bản ghi cuộc gọi, xem Message.Call.
	MessageCall MessageType = "call"
	// MessageDeletion: sự kiện thu hồi / xoá một tin đã gửi, xem Message.Deletion.
	MessageDeletion MessageType = "deletion"
)

func (m Message) Valid() bool {
	return m.ID != "" && m.AccountID != "" && m.ThreadID != "" && m.SenderID != "" && !m.OccurredAt.IsZero()
}
