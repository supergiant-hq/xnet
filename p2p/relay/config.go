package relay

import (
	"fmt"
	"net"

	"github.com/supergiant-hq/xnet/p2p"
	udpc "github.com/supergiant-hq/xnet/udp/client"
	udps "github.com/supergiant-hq/xnet/udp/server"
)

// Relay Server Config
type Config struct {
	// Debug Mode
	Debug bool
	// Listen Address
	Addr *net.UDPAddr
	// Broker Address
	BrokerAddr *net.UDPAddr
	// Broker Validation Token
	BrokerToken string

	udpsConfig udps.Config
	udpcConfig udpc.Config
}

func (c *Config) init() (err error) {
	c.udpsConfig = udps.Config{
		Tag:  "Relay",
		Addr: c.Addr,
	}

	c.udpcConfig = udpc.Config{
		Tag:            "Relay",
		ServerAddr:     c.BrokerAddr,
		ConnectTries:   0,
		ReconnectTries: 0,

		Token: c.BrokerToken,
		Data: map[string]string{
			p2p.KEY_PORT: fmt.Sprintf("%d", c.Addr.Port),
		},
	}

	return
}
