package udp

import (
	"fmt"
	"sync"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"

	"github.com/google/uuid"
	"github.com/lucas-clemente/quic-go"
	"github.com/sirupsen/logrus"
)

// On New Stream Handler
type StreamHandler func(Client, *Stream)

// Stream holds the configuration related to a single Stream
type Stream struct {
	// Stream ID
	Id      string
	channel *network.Channel
	// Stream Metadata
	Data        map[string]string
	initialized bool

	// Exit Channel
	Exit chan bool
	// Closed Status
	Closed bool
	mutex  sync.Mutex
}

// HandleStream wraps the quic.Stream and provides useful functionality to manage it
func HandleStream(logger *logrus.Logger, stream quic.Stream, unmarshalers []network.ChannelUnmarshaler) (cstream *Stream, err error) {
	channel := network.NewChannel(logger, stream, unmarshalers)

	msg, err := channel.Read(false)
	if err != nil {
		stream.Close()
		return
	}

	msgData, ok := msg.Body.(*model.StreamConnectionData)
	if !ok {
		err = fmt.Errorf("invalid connection message")
		stream.Close()
		return
	}

	rmsg, err := msg.GenReply(
		model.MessageTypeStreamConnectionStatus,
		&model.StreamConnectionStatus{
			Id:      msgData.Id,
			Status:  true,
			Message: "Accepted",
		},
	)
	if err != nil {
		return
	}
	if _, err = channel.Send(rmsg); err != nil {
		return
	}

	cstream = NewStreamFromData(msgData, channel)

	return
}

// Wraps a channel into a Stream and initializes it
func NewStream(data map[string]string, channel *network.Channel) (s *Stream, err error) {
	return newStream(uuid.NewString(), data, channel, true)
}

// Wraps a channel into a Stream and does not initialize it
// This function should be used when the stream is already initialized
func NewStreamFromData(data *model.StreamConnectionData, channel *network.Channel) *Stream {
	s, _ := newStream(data.Id, data.Data, channel, false)
	return s
}

func newStream(id string, data map[string]string, channel *network.Channel, shouldInitialize bool) (s *Stream, err error) {
	stream := &Stream{
		Id:          id,
		channel:     channel,
		Data:        data,
		initialized: !shouldInitialize,

		Exit: make(chan bool, 1),
	}

	if shouldInitialize {
		if err = stream.initialize(); err != nil {
			return
		}
	}

	s = stream
	return
}

func (s *Stream) initialize() (err error) {
	go s.channel.Read(false)

	msg := network.NewMessageWithAck(
		model.MessageTypeStreamConnectionData,
		&model.StreamConnectionData{
			Id:   s.Id,
			Data: s.Data,
		},
		network.RequestTimeout,
	)
	rmsg, err := s.channel.Send(msg)
	if err != nil {
		return
	}
	resData := rmsg.Body.(*model.StreamConnectionStatus)
	if !resData.Status {
		err = fmt.Errorf(resData.Message)
		return
	}

	s.initialized = true

	return
}

// Returns the channel associated with the Stream
func (s *Stream) Channel() *network.Channel {
	return s.channel
}

// Returns the Raw Stream
func (s *Stream) Stream() quic.Stream {
	return s.channel.Stream()
}

// Closes the Stream
func (s *Stream) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Closed {
		return
	}

	s.channel.Close()
	select {
	case s.Exit <- true:
	default:
	}
	s.Closed = true
}

// Stringify
func (s *Stream) String() string {
	return fmt.Sprintf("id(%s) initialized(%v)", s.Id, s.initialized)
}
