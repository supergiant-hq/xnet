package tun

import (
	"fmt"

	"os/exec"

	"github.com/songgao/water"
)

// Open TunDevice on Darwin
func (td *TunDevice) Open(config TunConfig) (err error) {
	if td.Active {
		return fmt.Errorf("tun: already active")
	}

	tun, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return fmt.Errorf("tun: activation error: %v", err)
	}

	// fmt.Println(config.CIDR.String(), config.CIDR.IP.String(), config.IP.String())

	// if err = exec.Command("/sbin/ifconfig", tun.Name(), config.CIDR.String(), config.CIDR.IP.String()).Run(); err != nil {
	// 	err = fmt.Errorf("tun: failed to run 'ifconfig': %s", err)
	// 	return
	// }
	if err = exec.Command("/sbin/ifconfig", tun.Name(), config.IP.String(), config.IP.String()).Run(); err != nil {
		return fmt.Errorf("tun: failed to run 'ifconfig': %s", err)
	}
	if err = exec.Command("/sbin/route", "-n", "add", "-net", config.CIDR.String(), "-interface", tun.Name()).Run(); err != nil {
		return fmt.Errorf("tun: failed to run 'route add': %s", err)
	}
	if err = exec.Command("/sbin/ifconfig", tun.Name(), "mtu", fmt.Sprintf("%d", config.MTU)).Run(); err != nil {
		return fmt.Errorf("tun: failed to run 'ifconfig': %s", err)
	}
	// if err = exec.Command("/sbin/ifconfig", tun.Name(), "up").Run(); err != nil {
	// 	err = fmt.Errorf("tun: failed to run 'ifconfig': %s", err)
	// 	return
	// }

	td.initTunDevice(&config, tun)
	fmt.Printf("tun: name(%s) active\n", td.Name)
	return
}
