package tun

import (
	"fmt"
	"net"
)

// Tun Config
type TunConfig struct {
	// Maximum transmission unit
	MTU int
	// IP Address of the Device
	IP net.IP
	// CIDR network of the Device
	CIDR *net.IPNet
}

// Create Tun Config
func NewTunConfig(mtu int, address string) (config TunConfig, err error) {
	ip, cidr, err := net.ParseCIDR(address)
	if err != nil {
		return TunConfig{}, fmt.Errorf("[tunconfig] err with %v", err)
	}
	config = TunConfig{
		MTU:  mtu,
		IP:   ip,
		CIDR: cidr,
	}
	return
}

// Stringify
func (tc *TunConfig) String() string {
	return fmt.Sprintf("%v", *tc)
}
