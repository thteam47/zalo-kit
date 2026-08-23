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
