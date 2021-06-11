package p2pc

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/p2p"
	"github.com/supergiant-hq/xnet/udp"
	udpc "github.com/supergiant-hq/xnet/udp/client"

	"github.com/sirupsen/logrus"
)

const (
	RELAY_CONNECT_TRIES      = 3
	RELAY_RECONNECT_TRIES    = 4
	RELAY_PEER_AWAIT_TIMEOUT = time.Second * 20
)

// Relay Connection
type relayConn struct {
	conn   *Connection
	addr   *net.UDPAddr
	client *udpc.Client

	connected bool
	exit      chan bool
	closed    bool
	mutex     sync.Mutex
	log       *logrus.Entry
}

func (c *Connection) newRelayConn() (conn *relayConn, err error) {
	addr, err := net.ResolveUDPAddr("udp", c.relayAddr)
	if err != nil {
		return
	}

	conn = &relayConn{
		conn: c,
		addr: addr,
		exit: make(chan bool, 1),
		log:  c.log,
	}
	return
}

func (c *relayConn) connect() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err = c.connectRelayServer(); err != nil {
		return
	}

	if err = c.awaitPeer(); err != nil {
		c.client.Close(0, fmt.Sprintf("Connect failed: %v", err.Error()))
		c.client = nil
		return
	}

	return
}

func (c *relayConn) connectRelayServer() (err error) {
	c.log.Infoln("Connecting to relay: ", c.conn.relayAddr)
	client, err := udpc.New(
		c.log.Logger,
		udpc.Config{
			Tag:            fmt.Sprintf("RELAY-%s", c.conn.id),
			ServerAddr:     c.addr,
			ConnectTries:   RELAY_CONNECT_TRIES,
			ReconnectTries: RELAY_RECONNECT_TRIES,

			TLS:  c.conn.mgr.client.Cfg.TLS.Clone(),
			Quic: c.conn.mgr.client.Cfg.Quic.Clone(),

			Token: c.conn.mgr.client.Cfg.Token,
			Data: map[string]string{
				p2p.KEY_CONNECTION_ID: c.conn.id,
			},
		},
	)
	if err != nil {
		return
	}

	if err = client.Connect(); err != nil {
		return
	}

	client.SetStreamHandler(c.conn.mgr.incomingStreamHandler)

	c.client = client

	go func() {
		<-c.client.Exit
		c.close()
	}()

	return
}

func (c *relayConn) awaitPeer() (err error) {
	msg := network.NewMessageWithAck(model.MessageTypeP2PRelayAwait, &model.NoDataMessage{}, RELAY_PEER_AWAIT_TIMEOUT)
	_, err = c.client.Send(msg)
	if err != nil {
		c.log.Errorln("Peer await connection timeout:", err.Error())
		return
	}
	c.log.Infoln("Peer connected to relay")

	c.connected = true

	return
}

func (c *relayConn) openStream(metadata map[string]string, data map[string]string) (stream *udp.Stream, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected {
		err = fmt.Errorf("not connected")
		return
	}

	msg := network.NewMessageWithAck(
		model.MessageTypeP2PRelayOpenStream,
		&model.P2PRelayOpenStream{
			Metadata: metadata,
			Data:     data,
		},
		network.ConnectionTimeout,
	)
	rmsg, err := c.client.Send(msg)
	if err != nil {
		return
	}

	streamInfo := rmsg.Body.(*model.P2PRelayStreamInfo)
	if !streamInfo.Status {
		err = fmt.Errorf(streamInfo.Message)
		return
	}

	return c.client.GetStream(streamInfo.Id)
}

func (c *relayConn) close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return
	}

	if c.client != nil {
		c.client.Close(0, "Close Called")
		c.client = nil
	}

	select {
	case c.exit <- true:
	default:
	}
	c.connected = false
	c.closed = true
}
