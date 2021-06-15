package network

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/supergiant-hq/xnet/network/model"

	"github.com/lucas-clemente/quic-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// Unmarshals protobuf encoded data depending on the MessageType
type ChannelUnmarshaler func(mtype MessageType) (msg proto.Message, err error)

// Channel
type Channel struct {
	unmarshalers []ChannelUnmarshaler

	stream quic.Stream
	rmutex sync.Mutex
	wmutex sync.Mutex

	acks   map[string]chan *Message
	amutex sync.Mutex

	log *logrus.Entry
}

// Create a new channel
func NewChannel(log *logrus.Logger, stream quic.Stream, unmarshalers []ChannelUnmarshaler) *Channel {
	return &Channel{
		unmarshalers: unmarshalers,
		stream:       stream,
		acks:         make(map[string]chan *Message),
		log:          log.WithField("prefix", "CHANNEL"),
	}
}

// Returns the channel's stream
func (c *Channel) Stream() quic.Stream {
	return c.stream
}

func (c *Channel) unmarshalData(mtype MessageType, bytes []byte) (data proto.Message, err error) {
	for _, unmarshaler := range c.unmarshalers {
		if data, err = unmarshaler(mtype); err == nil {
			err = proto.Unmarshal(bytes, data)
			break
		}
	}
	return
}

// Read single or multiple messages from the channel stream
func (c *Channel) Read(multiple bool) (msg *Message, err error) {
	err = fmt.Errorf("panic")

	defer func() {
		recover()
	}()

	c.rmutex.Lock()
	defer c.rmutex.Unlock()

	for {
		var (
			messageHeaderLengthBytes = make([]byte, messageHeaderSize)
			messageHeaderLength      uint32
			messageHeaderBytes       []byte
			messageHeader            = &model.HeaderData{}
			messageBodyBytes         []byte
			messageBody              proto.Message
		)

		if _, err = io.ReadFull(c.stream, messageHeaderLengthBytes); err != nil {
			return
		}
		messageHeaderLength = binary.BigEndian.Uint32(messageHeaderLengthBytes)
		messageHeaderBytes = make([]byte, messageHeaderLength)
		if _, err = io.ReadFull(c.stream, messageHeaderBytes); err != nil {
			return
		}
		if err = proto.Unmarshal(messageHeaderBytes, messageHeader); err != nil {
			return
		}

		messageBodyBytes = make([]byte, messageHeader.ContentLength)
		if _, err = io.ReadFull(c.stream, messageBodyBytes); err != nil {
			return
		}
		if messageBody, err = c.unmarshalData(MessageType(messageHeader.ContentType), messageBodyBytes); err != nil {
			return
		}

		msg = &Message{
			Ctx: MessageContext{
				Id:   messageHeader.Id,
				Ack:  messageHeader.Type == model.HeaderData_ACK,
				Type: MessageType(messageHeader.ContentType),
			},
			Body: messageBody,
		}

		c.log.Debugf("<- id(%v) of type(%s) with ack(%v)", msg.Ctx.Id, msg.Ctx.Type, msg.Ctx.Ack)

		if messageHeader.Type == model.HeaderData_ACKR {
			c.amutex.Lock()
			if ch, ok := c.acks[msg.Ctx.Id]; ok {
				select {
				case ch <- msg:
				default:
				}
			}
			c.amutex.Unlock()
			if multiple {
				continue
			} else {
				break
			}
		}

		break
	}

	return
}

func (c *Channel) write(msg *Message) (err error) {
	defer recover()

	msg.init()

	var (
		messageType              model.HeaderData_Type
		messageHeaderLengthBytes []byte = make([]byte, messageHeaderSize)
		messageHeaderBytes       []byte
		messageBodyBytes         []byte

		send []byte
	)

	if messageBodyBytes, err = proto.Marshal(msg.Body); err != nil {
		return
	}

	if msg.Ctx.Ack {
		messageType = model.HeaderData_ACK
	} else if msg.Ctx.Ackr {
		messageType = model.HeaderData_ACKR
	} else {
		messageType = model.HeaderData_NORMAL
	}

	messageHeader := model.HeaderData{
		Type:          messageType,
		Id:            msg.Ctx.Id,
		ContentType:   string(msg.Ctx.Type),
		ContentLength: uint32(len(messageBodyBytes)),
	}
	if messageHeaderBytes, err = proto.Marshal(&messageHeader); err != nil {
		return
	}

	binary.BigEndian.PutUint32(messageHeaderLengthBytes, uint32(len(messageHeaderBytes)))

	send = messageHeaderLengthBytes
	send = append(send, messageHeaderBytes[:]...)
	send = append(send, messageBodyBytes[:]...)

	c.wmutex.Lock()
	_, err = c.stream.Write(send)
	c.wmutex.Unlock()
	if err != nil {
		return
	}

	c.log.Debugf("-> id(%v) of type(%s) with ack(%v)", msg.Ctx.Id, msg.Ctx.Type, msg.Ctx.Ack)
	return
}

// Send a message through the channel stream
func (c *Channel) Send(msg *Message) (rmsg *Message, err error) {
	var resChan chan *Message
	if msg.Ctx.Ack {
		c.amutex.Lock()
		resChan = make(chan *Message, 1)
		c.acks[msg.Ctx.Id] = resChan
		c.amutex.Unlock()
	}

	defer func() {
		recover()
		if msg.Ctx.Ack {
			c.amutex.Lock()
			delete(c.acks, msg.Ctx.Id)
			c.amutex.Unlock()
		}
	}()

	if err = c.write(msg); err != nil {
		return
	}

	if msg.Ctx.Ack {
		select {
		case resMsg, ok := <-resChan:
			if !ok {
				err = fmt.Errorf("response channel closed")
				break
			}
			rmsg = resMsg
		case <-time.After(msg.Opt.Timeout):
			err = fmt.Errorf("request timeout")
		}
	}

	return
}

// If the channel is not actively read, this function sends and reads a single message
// This method SHOULD NOT BE USED if the channel is actively being read. Just use the Send function.
func (c *Channel) SendAndRead(msg *Message) (rmsg *Message, err error) {
	defer recover()
	go c.Read(false)
	return c.Send(msg)
}

// Closes the channel
func (c *Channel) Close() {
	c.amutex.Lock()
	defer c.amutex.Unlock()

	for id, ch := range c.acks {
		close(ch)
		delete(c.acks, id)
	}

	c.stream.Close()
}
