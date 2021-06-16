package network

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const (
	messageHeaderSize uint32 = 4
)

// Message Type
type MessageType string

// Stores the context of a message
type MessageContext struct {
	// Message ID
	Id string
	// Message needs an Acknowledgement
	Ack bool
	// Message is Acknowledgement Reply
	Ackr bool
	// Message Type
	Type MessageType
}

func (c *MessageContext) init() {
	if len(c.Id) == 0 {
		c.Id = uuid.New().String()
	}
}

// Message Options
type MessageOptions struct {
	// Duration after which the Message Request Timesout
	Timeout time.Duration
}

func (o *MessageOptions) init() {
	if o.Timeout == 0 {
		o.Timeout = RequestTimeout
	}
}

// Message
type Message struct {
	// Message Context
	Ctx MessageContext
	// Message Options
	Opt MessageOptions
	// Message Body
	Body proto.Message
}

// New Message Options
type NewMessageOptions struct {
	// Message requires an ACK
	Ack bool
	// Message Timeout Duration
	Timeout time.Duration
}

// New Message with default options
func NewMessage(mtype MessageType, mbody proto.Message) (msg *Message) {
	return NewMessageWithOptions(mtype, mbody, NewMessageOptions{})
}

// New Message which requires an ACK
func NewMessageWithAck(mtype MessageType, mbody proto.Message, timeout time.Duration) (msg *Message) {
	return NewMessageWithOptions(mtype, mbody, NewMessageOptions{
		Ack:     true,
		Timeout: timeout,
	})
}

// New Message with custom options
func NewMessageWithOptions(mtype MessageType, mbody proto.Message, nopt NewMessageOptions) (msg *Message) {
	ctx := MessageContext{
		Ack:  nopt.Ack,
		Type: mtype,
	}

	opt := MessageOptions{
		Timeout: nopt.Timeout,
	}

	msg = &Message{
		Ctx:  ctx,
		Opt:  opt,
		Body: mbody,
	}

	msg.init()

	return
}

func (m *Message) init() {
	m.Ctx.init()
	m.Opt.init()
}

// Generate message reply
func (m *Message) GenReply(mtype MessageType, mbody proto.Message) (msg *Message, err error) {
	if !m.Ctx.Ack {
		err = ErrorGenRes
		return
	}

	msg = NewMessage(mtype, mbody)
	msg.Ctx.Id = m.Ctx.Id
	msg.Ctx.Ackr = true
	msg.init()

	return
}

// Stringify
func (m *Message) String() string {
	return fmt.Sprintf("%+v", m.Ctx)
}
