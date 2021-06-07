package brokerc

import (
	p2pc "github.com/supergiant-hq/xnet/p2p/client"
	udpc "github.com/supergiant-hq/xnet/udp/client"
)

// Broker Client Config
type Config struct {
	// Debug Mode
	Debug bool
	// UDP Client Config
	UdpcConfig udpc.Config
	// P2P Client Manager Config
	P2PConfig p2pc.Config
}
