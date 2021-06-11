package p2pc

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/udp"
)

// Message Handler
type MessageHandler func(mc *MessageStream, msg *network.Message)

// Message Stream
type MessageStream struct {
	Conn   *Connection
	stream *udp.Stream

	// Exit channel
	Exit chan bool
	// Message Stream closed
	Closed bool
	mutex  sync.Mutex
	log    *logrus.Entry
}

// Create a MessageStream
func NewMessageStream(conn *Connection, stream *udp.Stream) *MessageStream {
	return &MessageStream{
		Conn:   conn,
		stream: stream,

		Exit: make(chan bool, 1),
		log:  conn.log.WithField("prefix", fmt.Sprintf("%s:MessageStream", conn.id)),
	}
}

// Listen for messages
func (mc *MessageStream) Listen(mh MessageHandler) (err error) {
	if mc.Closed {
		err = fmt.Errorf("message stream closed")
		return
	} else if mh == nil {
		err = fmt.Errorf("MessageHandler not provided")
		return
	}

	go func() {
		go func() {
			recover()
			mc.Close()
		}()

		for {
			msg, err := mc.stream.Channel().Read(true)
			if err != nil {
				mc.log.Warnln("Closing message stream:", err.Error())
				mc.Close()
				return
			}

			go mh(mc, msg)
		}
	}()

	return
}

// Send a message
func (mc *MessageStream) Send(msg *network.Message) (rmsg *network.Message, err error) {
	return mc.stream.Channel().Send(msg)
}

// If the stream is not actively read, this function sends and reads a single message
// This method SHOULD NOT BE USED if the stream is actively being read. Just use the Send function.
func (mc *MessageStream) SendAndRead(msg *network.Message) (rmsg *network.Message, err error) {
	return mc.stream.Channel().SendAndRead(msg)
}

// Close Message Stream
func (mc *MessageStream) Close() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.Closed {
		return
	}

	mc.stream.Close()

	select {
	case mc.Exit <- true:
	default:
	}
	mc.Closed = true
}
