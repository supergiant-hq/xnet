package p2pc

import (
	"fmt"
	"sync"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/p2p"
	"github.com/supergiant-hq/xnet/tun"
	"github.com/supergiant-hq/xnet/udp"
	"github.com/supergiant-hq/xnet/util"

	"github.com/sirupsen/logrus"
)

// P2P Client Connection
type Connection struct {
	mgr *Manager
	// Client ID
	ClientId string

	initiator bool
	id        string
	mode      p2p.ConnectionMode
	peer      *peer

	p2pConn   *p2pConn
	relayAddr string
	relayConn *relayConn

	// Exit Channel
	Exit chan bool
	// Closed Status
	Closed bool
	mutex  sync.Mutex
	log    *logrus.Entry
}

func createConnection(log *logrus.Logger, mgr *Manager, peerId string, mode p2p.ConnectionMode) (c *Connection, err error) {
	interfaceAddrs, err := tun.GetInterfaceAddresses()
	if err != nil {
		return
	}
	interfaceIPs := []string{}
	for _, addr := range interfaceAddrs {
		interfaceIPs = append(interfaceIPs, fmt.Sprintf("%s:%d", addr.String(), mgr.client.Addr.Port))
	}
	relayAddress := ""

	if mode == p2p.ConnectionModeRelay {
		relayAddress, err = mgr.getNearestRelay()
		if err != nil {
			return
		}
	}

	msg := network.NewMessageWithAck(
		model.MessageTypeP2PConnectionRequest,
		&model.P2PConnectionRequest{
			Mode:         string(mode),
			RelayAddress: relayAddress,
			Peer: &model.P2PPeerData{
				Id:        peerId,
				Address:   mgr.client.Addr.String(),
				Addresses: util.RemoveDuplicatesFromSlice(interfaceIPs),
			},
		},
		p2p.ConnectionTimeout,
	)
	mres, err := mgr.client.Send(msg)
	if err != nil {
		return
	}

	connData := mres.Body.(*model.P2PConnectionData)
	if !connData.Status {
		err = fmt.Errorf(connData.Message)
		return
	}

	peer, err := newPeer(connData.Peer)
	if err != nil {
		return
	}
	c = &Connection{
		mgr:      mgr,
		ClientId: mgr.client.Id,

		initiator: true,
		id:        connData.Id,
		mode:      mode,
		peer:      peer,
		relayAddr: relayAddress,

		Exit: make(chan bool, 1),
		log:  log.WithField("prefix", fmt.Sprintf("P2P-CONN-%s", connData.Id)),
	}
	mgr.log.Infof("Peer accepted connection request: %s", c.String())

	if err = c.connect(); err != nil {
		return
	}

	return
}

func acceptConnection(log *logrus.Logger, mgr *Manager, connData *model.P2PConnectionRequest) (c *Connection, err error) {
	peer, err := newPeer(connData.Peer)
	if err != nil {
		return
	}

	c = &Connection{
		mgr:      mgr,
		ClientId: mgr.client.Id,

		initiator: false,
		id:        connData.Id,
		mode:      p2p.ConnectionMode(connData.Mode),
		peer:      peer,
		relayAddr: connData.RelayAddress,

		Exit: make(chan bool, 1),
		log:  log.WithField("prefix", fmt.Sprintf("P2P-CONN-%s", connData.Id)),
	}
	mgr.log.Infof("Accepted peer connection request: %s", c.String())

	go func() {
		if err := c.connect(); err != nil {
			mgr.log.Errorf("Error waiting for peer connection: %s", err.Error())
		}
	}()

	return
}

func (c *Connection) connect() (err error) {
	defer func() {
		if err != nil {
			c.log.Errorln("Connect Error:", err.Error())
		}
	}()

	switch c.mode {
	case p2p.ConnectionModeP2P:
		c.p2pConn = c.newP2PConn()
		if err = c.p2pConn.connect(); err != nil {
			c.p2pConn.close()
			c.p2pConn = nil
			return
		}

		go func() {
			<-c.p2pConn.exit
			c.close("Exited")
		}()

	case p2p.ConnectionModeRelay:
		c.relayConn, err = c.newRelayConn()
		if err != nil {
			return
		}
		if err = c.relayConn.connect(); err != nil {
			c.relayConn.close()
			c.relayConn = nil
			return
		}

		go func() {
			<-c.relayConn.exit
			c.close("Exited")
		}()

	default:
		err = fmt.Errorf("invalid connection mode: %v", c.mode)
		c.close("Invalid mode")
		return
	}

	return
}

// Open Stream to Peer
func (c *Connection) OpenStream(data map[string]string) (stream *udp.Stream, err error) {
	if !c.IsConnected() {
		err = fmt.Errorf("not connected")
		return
	}

	switch c.mode {
	case p2p.ConnectionModeP2P:
		return c.p2pConn.OpenStream(data)
	case p2p.ConnectionModeRelay:
		return c.relayConn.OpenStream(data)
	}
	return
}

func (c *Connection) notifyNewConnection() {
	go c.mgr.connectionHandler(c)
}

// If Connection is active
func (c *Connection) IsConnected() bool {
	switch c.mode {
	case p2p.ConnectionModeP2P:
		return c.p2pConn != nil && c.p2pConn.connected
	case p2p.ConnectionModeRelay:
		return c.relayConn != nil && c.relayConn.connected
	default:
		return false
	}
}

func (c *Connection) close(reason string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.Closed {
		return
	}

	if c.p2pConn != nil {
		c.p2pConn.close()
		c.p2pConn = nil
	}

	if c.relayConn != nil {
		c.relayConn.close()
		c.relayConn = nil
	}

	select {
	case c.Exit <- true:
	default:
	}
	c.Closed = true

	c.log.Warnf("Connection closed: %s", reason)
}

// Stringify
func (c *Connection) String() string {
	return fmt.Sprintf("id(%s) with mode(%v) peer(%v) closed(%v)", c.id, c.mode, c.peer.id, c.Closed)
}
