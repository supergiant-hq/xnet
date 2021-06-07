package udps

import (
	"crypto/tls"
	"math"
	"net"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"

	"github.com/lucas-clemente/quic-go"
)

// UDP Server Config
type Config struct {
	// Log Tag
	Tag string
	// Listen Address
	Addr *net.UDPAddr
	// Protobuf Unmarshaler
	Unmarshaler network.ChannelUnmarshaler
	// Use QUIC Datagrams
	Datagrams bool

	// TLS Config
	TLS *tls.Config
	// QUIC Config
	Quic *quic.Config

	managed bool
}

func (c *Config) init(managed bool) (err error) {
	if len(c.Tag) == 0 {
		c.Tag = "Default"
	}

	if c.Addr == nil {
		if c.Addr, err = net.ResolveUDPAddr("udp", ":10000"); err != nil {
			return
		}
	}

	c.TLS = network.GenerateTLSConfig()
	c.Quic = network.GenerateQuicConfig(c)
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
