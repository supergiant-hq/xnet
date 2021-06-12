package udp

import (
	"github.com/supergiant-hq/xnet/network"
)

// Client Interface stipulates functions which should be present in a client
type Client interface {
	// ID
	ID() string
	// Stringify
	String() string
	// Closes the client
	Close(code int, reason string)

	// Sends a Message to the Server
	Send(msg *network.Message) (rmsg *network.Message, err error)
	// Opens a Stream to the Server
	OpenStream(metadata map[string]string, data map[string]string) (stream *Stream, err error)
	// Closes a Stream
	CloseStream(id string)
}
