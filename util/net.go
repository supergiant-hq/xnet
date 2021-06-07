// Utilities for "net"
package util

import (
	"fmt"
	"math/rand"
	"net"
)

// Generates a UDP Address with localhost:random-port
func GetRandomUDPAddress(min, max int) (addr *net.UDPAddr, err error) {
	return net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", min+rand.Intn(max-min)))
}
