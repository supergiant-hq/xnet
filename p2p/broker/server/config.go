package brokers

import udps "github.com/supergiant-hq/xnet/udp/server"

// Broker Server Config
type Config struct {
	// Debug Mode
	Debug bool
	// UDP Server Config
	UdpsConfig udps.Config
}
