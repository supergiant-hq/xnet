package tun

import (
	"fmt"
	"net"

	"github.com/songgao/water"
)

// Get Device Network Interfaces
func GetInterfaceAddresses() (addrs []net.IP, err error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return
	}

	for _, ifc := range ifs {
		iaddrs, err := ifc.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range iaddrs {
			ip, _, _ := net.ParseCIDR(addr.String())
			if ip.To4() != nil {
				addrs = append(addrs, ip.To4())
			}
		}
	}

	return
}

// Tun Device
type TunDevice struct {
	// Config
	Config TunConfig
	// Name
	Name string
	// IO Device
	Device *water.Interface
	// Active Status
	Active bool
}

func (td *TunDevice) initTunDevice(config *TunConfig, wi *water.Interface) {
	td.Config = *config
	td.Name = wi.Name()
	td.Device = wi
	td.Active = true
}

// Close Device
func (td *TunDevice) Close() {
	td.Device.Close()
	td.Name = ""
	td.Device = nil
	td.Active = false
}

// Stringify
func (td *TunDevice) String() string {
	return fmt.Sprintf("%v", *td)
}
