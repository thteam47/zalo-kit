package client

import (
	"errors"
	"github.com/thteam47/zago"
	"github.com/thteam47/zalo-kit/inbound"
	"testing"
)

func TestRemoveReactionValidationAndSession(t *testing.T) {
	c := &Client{}
	ref := MessageRef{MsgID: "123", CliMsgID: "456"}
	for _, kind := range []inbound.ThreadType{inbound.ThreadDirect, inbound.ThreadGroup} {
		if err := c.RemoveReaction("thread", kind, ref); !errors.Is(err, ErrSessionInvalid) {
			t.Fatalf("valid removal must reach session guard (%s): %v", kind, err)
		}
	}
	for _, err := range []error{
		c.RemoveReaction("", inbound.ThreadDirect, ref),
		c.RemoveReaction("t", inbound.ThreadDirect, MessageRef{MsgID: "123"}),
		c.RemoveReaction("t", inbound.ThreadDirect, MessageRef{CliMsgID: "456"}),
		c.RemoveReaction("t", inbound.ThreadType("invalid"), ref),
	} {
		if err == nil || errors.Is(err, ErrSessionInvalid) {
			t.Fatalf("invalid target must fail before session access: %v", err)
		}
	}
}

type reactionCapture struct {
	message      *zago.MessageObject
	icon, thread string
	kind         zago.ThreadType
	reactionType int
	err          error
}

func (c *reactionCapture) SendReaction(m *zago.MessageObject, icon, thread string, kind zago.ThreadType, reactionType int) (any, error) {
	c.message, c.icon, c.thread, c.kind, c.reactionType = m, icon, thread, kind, reactionType
	return nil, c.err
}

func TestRemoveReactionProviderContract(t *testing.T) {
	for _, kind := range []inbound.ThreadType{inbound.ThreadDirect, inbound.ThreadGroup} {
		capture := &reactionCapture{}
		if err := removeReaction(capture, "thread", kind, MessageRef{MsgID: "123", CliMsgID: "456", MsgType: "webchat"}); err != nil {
			t.Fatal(err)
		}
		if capture.icon != "" || capture.reactionType != -1 || capture.thread != "thread" || capture.kind != zaloThreadType(kind) || capture.message.Get("msgId") != "123" || capture.message.Get("cliMsgId") != "456" {
			t.Fatalf("invalid removal wire arguments: %+v", capture)
		}
		capture.err = errors.New("session expired")
		if err := removeReaction(capture, "thread", kind, MessageRef{MsgID: "123", CliMsgID: "456"}); !errors.Is(err, capture.err) {
			t.Fatalf("lost provider error: %v", err)
		}
	}
}
