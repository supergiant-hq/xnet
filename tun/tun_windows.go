package tun

import (
	"fmt"
	"net"

	"os/exec"

	"github.com/songgao/water"
)

// Open TunDevice on Windows
// TODO Implement and Test
func (td *TunDevice) Open(config TunConfig) (err error) {
	if td.Active {
		return fmt.Errorf("[tun] already active")
	}

	tun, err := water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			ComponentID:   "tap0901",
			InterfaceName: "watertun0901",
			Network:       config.CIDR.String(),
		},
	})
	if err != nil {
		return fmt.Errorf("[tun] activation error: %v", err)
	}

	if err = exec.Command(
		`C:\Windows\System32\netsh.exe`, "interface", "ipv4", "set", "address",
		fmt.Sprintf("name=%s", tun.Name()),
		"source=static",
		// fmt.Sprintf("addr=%s", config.CIDR.IP),
		fmt.Sprintf("addr=%s", config.IP),
		fmt.Sprintf("mask=%s", net.IP(config.CIDR.Mask)),
		"gateway=none",
	).Run(); err != nil {
		return fmt.Errorf("failed to run 'netsh' to set address: %s", err)
	}
	if err = exec.Command(
		`C:\Windows\System32\netsh.exe`, "interface", "ipv4", "set", "interface",
		tun.Name(),
		fmt.Sprintf("mtu=%d", config.MTU),
	).Run(); err != nil {
		return fmt.Errorf("failed to run 'netsh' to set MTU: %s", err)
	}
	if _, err = net.InterfaceByName(tun.Name()); err != nil {
		return fmt.Errorf("failed to find interface named %s: %v", tun.Name(), err)
	}

	td.initTunDevice(&config, tun)
	fmt.Printf("[tun][%s] active\n", td.Name)
	return
}
