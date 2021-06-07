package tun

import (
	"fmt"

	"os/exec"

	"github.com/songgao/water"
)

// Open TunDevice on Linux
func (td *TunDevice) Open(config TunConfig) (err error) {
	if td.Active {
		return fmt.Errorf("[tun] already active")
	}

	tun, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return fmt.Errorf("[tun] activation error: %v", err)
	}

	if err = exec.Command("/sbin/ip", "link", "set", "dev", tun.Name(), "mtu", fmt.Sprintf("%d", config.MTU)).Run(); err != nil {
		return fmt.Errorf("failed to run 'ifconfig': %s", err)
	}
	if err = exec.Command("/sbin/ip", "addr", "add", config.IP.String(), "dev", tun.Name()).Run(); err != nil {
		return fmt.Errorf("failed to run 'ifconfig': %s", err)
	}
	if err = exec.Command("/sbin/ip", "addr", "add", config.CIDR.String(), "dev", tun.Name()).Run(); err != nil {
		return fmt.Errorf("failed to run 'ifconfig': %s", err)
	}
	if err = exec.Command("/sbin/ip", "link", "set", "dev", tun.Name(), "up").Run(); err != nil {
		err = fmt.Errorf("failed to run 'ifconfig': %s", err)
		return
	}

	td.initTunDevice(&config, tun)
	fmt.Printf("[tun][%s] active\n", td.Name)
	return
}
