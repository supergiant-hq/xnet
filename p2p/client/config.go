package p2pc

import (
	"net"
)

// P2P Client Manager
type Config struct {
	// Relay Server Address
	// Providing an address (non-nil) will ensure that all relay connections from this client
	// will use the server with this address
	RelayAddr *net.UDPAddr
}
