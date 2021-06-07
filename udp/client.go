package udp

import "github.com/supergiant-hq/xnet/network"

// Client Interface stipulates functions which should be present in a client
type Client interface {
	ID() string
	String() string
	Close(code int, reason string)

	Send(msg *network.Message) (rmsg *network.Message, err error)
	OpenStream(data map[string]string) (stream *Stream, err error)
	CloseStream(id string)
}
