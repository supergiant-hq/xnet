package p2ps

import (
	"fmt"

	udps "github.com/supergiant-hq/xnet/udp/server"
)

// P2P Server Peer
type peer struct {
	id     string
	client *udps.Client
}

func newPeer(c *udps.Client) *peer {
	return &peer{
		id:     c.Id,
		client: c,
	}
}

// Stringify
func (p *peer) String() string {
	return fmt.Sprintf("id(%s) with address(%s)", p.id, p.client.Addr.String())
}
