package udpc

import (
	"crypto/tls"
	"fmt"
	"math"
	"net"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"

	"github.com/lucas-clemente/quic-go"
)

// UDP Client Config
type Config struct {
	// Log Tag
	Tag string
	// Server Address
	ServerAddr *net.UDPAddr
	// No. of times to try to connect
	ConnectTries int
	// No. of times to try to reconnect
	ReconnectTries int
	// Use QUID Datagram
	Datagrams   bool
	Unmarshaler network.ChannelUnmarshaler

	// Server Token
	Token string
	// Metadata
	Data map[string]string

	// TLS Config
	TLS *tls.Config
	// QUIC Config
	Quic *quic.Config

	managed bool
}

func (c *Config) init(managed bool) (err error) {
	if len(c.Tag) == 0 {
		c.Tag = "UDPC"
	}

	if c.Data == nil {
		c.Data = make(map[string]string)
	}

	if c.ServerAddr == nil {
		if c.ServerAddr, err = net.ResolveUDPAddr("udp", ":10000"); err != nil {
			return
		}
	}

	c.TLS = network.GenerateTLSConfig()
	c.Quic = network.GenerateQuicConfig(c)

	if len(c.Token) == 0 {
		err = fmt.Errorf("token cannot be empty")
		return
	}

	c.managed = managed

	return
}

// Maximum number of streams per connection
func (c *Config) MaxStreams() int16 {
	return int16(math.Pow(10, 3))
}

// Whether to use Datagram mode for QUIC
func (c *Config) UseDatagram() bool {
	return c.Datagrams
}

// Protobuf Unmarshalers
func (c *Config) Unmarshalers() (unmarshalers []network.ChannelUnmarshaler) {
	unmarshalers = []network.ChannelUnmarshaler{model.Unmarshal}
	if c.Unmarshaler != nil {
		unmarshalers = append(unmarshalers, c.Unmarshaler)
	}
	return
}
