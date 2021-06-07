package p2pc

import (
	"fmt"
	"net"

	"github.com/supergiant-hq/xnet/model"
)

type peer struct {
	id    string
	addr  *net.UDPAddr
	addrs []*net.UDPAddr
}

func newPeer(data *model.P2PPeerData) (p *peer, err error) {
	addr, err := net.ResolveUDPAddr("udp", data.Address)
	if err != nil {
		return
	}

	addrs := []*net.UDPAddr{}
	for _, raddr := range data.Addresses {
		addr, err := net.ResolveUDPAddr("udp", raddr)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}

	p = &peer{
		id:    data.Id,
		addr:  addr,
		addrs: addrs,
	}

	return
}

// Stringify
func (p *peer) String() string {
	return fmt.Sprintf("id(%s) with address(%s)", p.id, p.addr.String())
}
