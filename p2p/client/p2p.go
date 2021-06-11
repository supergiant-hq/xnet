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
	udps "github.com/supergiant-hq/xnet/udp/server"

	"github.com/sirupsen/logrus"
)

const (
	P2P_CONNECT_TRIES   = 3
	P2P_RECONNECT_TRIES = 3
)

type connectPeerContext struct {
	resultChan chan *udpc.Client
	mutex      sync.RWMutex
	completed  bool
}

// P2P Connection
type p2pConn struct {
	conn *Connection

	localClient      *udpc.Client
	remoteClientChan chan *udps.Client
	remoteClient     *udps.Client

	connected bool
	exit      chan bool
	closed    bool
	mutex     sync.Mutex
	log       *logrus.Entry
}

func (c *Connection) newP2PConn() *p2pConn {
	return &p2pConn{
		conn: c,

		remoteClientChan: make(chan *udps.Client, 1),
		exit:             make(chan bool, 1),
		log:              c.log,
	}
}

func (c *p2pConn) connect() (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	go c.punch(c.conn.mgr.client.UDPConn, c.conn.peer.addrs)

	if c.conn.initiator {
		connectContext := &connectPeerContext{
			resultChan: make(chan *udpc.Client),
		}
		for _, serverAddr := range c.conn.peer.addrs {
			go c.connectToPeer(serverAddr, connectContext)
		}

	loop:
		for {
			select {
			case client := <-connectContext.resultChan:
				connectContext.mutex.Lock()

				err = c.initClient(client)
				if err != nil {
					connectContext.mutex.Unlock()
					continue
				}

				go func() {
					<-client.Exit
					c.close()
				}()

				connectContext.completed = true
				connectContext.mutex.Unlock()

				break loop
			case <-time.After(network.ConnectionTimeout * P2P_CONNECT_TRIES):
				err = fmt.Errorf("connecting to peer timeout")
				break loop
			}
		}
	} else {
		if err = c.awaitRemoteConnection(); err != nil {
			return
		}
	}

	return
}

func (c *p2pConn) connectToPeer(serverAddr *net.UDPAddr, connectCtx *connectPeerContext) (client *udpc.Client, err error) {
	client, err = udpc.NewWithConnection(
		c.log.Logger,
		udpc.Config{
			Tag:            c.conn.id,
			ServerAddr:     serverAddr,
			ConnectTries:   P2P_CONNECT_TRIES,
			ReconnectTries: P2P_RECONNECT_TRIES,

			TLS:  c.conn.mgr.client.Cfg.TLS.Clone(),
			Quic: c.conn.mgr.client.Cfg.Quic.Clone(),

			Token: c.conn.id,
		},
		c.conn.mgr.client.Addr,
		c.conn.mgr.client.UDPConn,
	)
	if err != nil {
		return
	}

	client.SetCanConnectHandler(func(tries int) bool {
		connectCtx.mutex.RLock()
		defer connectCtx.mutex.RUnlock()
		return !connectCtx.completed && tries < P2P_CONNECT_TRIES
	})

	if err = client.Connect(); err != nil {
		return
	}

	connectCtx.mutex.RLock()
	defer connectCtx.mutex.RUnlock()
	select {
	case connectCtx.resultChan <- client:
		return
	default:
		client.Close(0, "Already connected using a different client")
	}

	return
}

func (c *p2pConn) initClient(client *udpc.Client) (err error) {
	msg := network.NewMessageWithAck(
		model.MessageTypeP2PClientInit,
		&model.NoDataMessage{},
		p2p.RequestTimeout,
	)
	rmsg, err := client.Send(msg)
	if err != nil {
		return
	}

	connStatus := rmsg.Body.(*model.P2PConnectionStatus)
	if !connStatus.Status {
		err = fmt.Errorf(connStatus.Message)
		return
	}

	client.SetStreamHandler(c.conn.mgr.incomingStreamHandler)

	c.localClient = client
	c.connected = true

	// if _, err = c.OpenStream(map[string]string{
	// 	p2p.KEY_STREAM_INTERNAL: "true",
	// }); err != nil {
	// 	c.log.Errorln("Peer open channel stream error:", err.Error())
	// 	return
	// }

	return
}

func (c *p2pConn) awaitRemoteConnection() (err error) {
	select {
	case remoteClient, ok := <-c.remoteClientChan:
		if !ok {
			err = fmt.Errorf("await peer channel closed")
			break
		}

		c.remoteClient = remoteClient
		c.connected = true

	case <-time.After(time.Minute / 2):
		err = fmt.Errorf("awaiting for peer timedout")
	}

	return
}

func (c *p2pConn) checkinRemoteClient(client *udps.Client) (err error) {
	if c.connected {
		err = fmt.Errorf("connection already open")
		return
	} else if c.remoteClient != nil {
		err = fmt.Errorf("another client is already connected")
		return
	}

	select {
	case c.remoteClientChan <- client:
	default:
	}

	return
}

func (c *p2pConn) punch(conn *net.UDPConn, addrs []*net.UDPAddr) {
	for {
		if c.connected || c.closed {
			return
		}

		if conn != nil {
			c.log.Debugf("Punching to ips(%v) for conn(%s)", addrs, c.conn.id)

			for _, addr := range addrs {
				conn.WriteTo([]byte("punch!"), addr)
			}
		}

		time.Sleep(time.Second * 1)
	}
}

func (c *p2pConn) openStream(metadata map[string]string, data map[string]string) (stream *udp.Stream, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected {
		err = fmt.Errorf("not connected")
		return
	}

	if c.conn.initiator {
		return c.localClient.OpenStream(metadata, data)
	} else {
		return c.remoteClient.OpenStream(metadata, data)
	}
}

func (c *p2pConn) close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return
	}

	if c.localClient != nil {
		c.localClient.Close(0, "Closed")
		c.localClient = nil
	}

	if c.remoteClient != nil {
		c.remoteClient.Close(0, "Closed")
		c.remoteClient = nil
	}

	select {
	case c.exit <- true:
	default:
	}
	c.connected = false
	c.closed = true
}
