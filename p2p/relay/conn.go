package relay

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/p2p"
	"github.com/supergiant-hq/xnet/udp"
	udps "github.com/supergiant-hq/xnet/udp/server"
	"github.com/supergiant-hq/xnet/util"

	"github.com/guiguan/caster"

	"github.com/sirupsen/logrus"
)

const (
	tickerDuration   = time.Second * 30
	awaitPeerTimeout = time.Second * 15
)

// Relay Server Connection
type Connection struct {
	server *Server
	id     string

	sourcePeer  *model.P2PPeerData
	targetPeer  *model.P2PPeerData
	peerConnBus *caster.Caster

	// Closed Status
	Closed      bool
	tickerTimer *time.Timer
	ticker      *util.Ticker
	mutex       sync.Mutex
	log         *logrus.Entry
}

// Create a new Connection
func (s *Server) NewConnection(data *model.P2PRelayConnectionData) (conn *Connection) {
	conn = &Connection{
		server: s,
		id:     data.Id,

		sourcePeer:  data.SourcePeer,
		targetPeer:  data.TargetPeer,
		peerConnBus: caster.New(context.Background()),

		log: s.log.WithField("prefix", fmt.Sprintf("RELAY-%s", data.Id)),
	}
	conn.ticker = util.NewTicker(tickerDuration, conn.handleTick)
	conn.tickerTimer = time.AfterFunc(tickerDuration, conn.ticker.Start)

	s.conns.Store(conn.id, conn)

	return
}

func (c *Connection) handleTick() {
	if c.Closed {
		c.stopTicker()
		return
	}

	var err error

	defer func() {
		if err != nil {
			c.stopTicker()
			c.Close()
		}
	}()

	msg := network.NewMessageWithAck(
		model.MessageTypeP2PConnectionStatus,
		&model.P2PConnectionStatus{
			Id: c.id,
		},
		network.RequestTimeout,
	)
	rmsg, err := c.server.udpClient.Send(msg)
	if err != nil {
		return
	}

	rdata := rmsg.Body.(*model.P2PConnectionStatus)
	if !rdata.Status {
		err = fmt.Errorf(rdata.Message)
		return
	}
}

func (c *Connection) getClientId(peerId string) string {
	return fmt.Sprintf("%s:%s", peerId, c.id)
}

func (c *Connection) getPeerClientId(client *udps.Client) (peerClientId string, err error) {
	if client.Id == c.getClientId(c.sourcePeer.Id) {
		peerClientId = c.getClientId(c.targetPeer.Id)
	} else if client.Id == c.getClientId(c.targetPeer.Id) {
		peerClientId = c.getClientId(c.sourcePeer.Id)
	} else {
		err = fmt.Errorf("invalid client id: something is horribly wrong")
	}
	return
}

func (c *Connection) getPeerClient(client *udps.Client) (*udps.Client, error) {
	peerClientId, err := c.getPeerClientId(client)
	if err != nil {
		return nil, err
	}
	return c.server.udpServer.GetClient(peerClientId)
}

func (c *Connection) awaitPeer(client *udps.Client) (err error) {
	// Notify the channel that this client is connected
	c.peerConnBus.TryPub(client.Id)

	// Check if the other peer is already connected
	if peerClient, _ := c.getPeerClient(client); peerClient != nil {
		return
	}

	// Await connection from the other peer for (awaitPeertimeout) nanoseconds
	ch, ok := c.peerConnBus.Sub(context.Background(), 1)
	if !ok {
		err = fmt.Errorf("error subscribing to broadcast channel")
		return
	}
	defer c.peerConnBus.Unsub(ch)

	peerId, err := c.getPeerClientId(client)
	if err != nil {
		return
	}
	timeNow := time.Now()
	endTime := timeNow.Add(awaitPeerTimeout)
	for endTime.Unix()-timeNow.Unix() > 0 {
		select {
		case id := <-ch:
			if id == peerId {
				return
			}
			timeNow = time.Now()
		case <-time.After(endTime.Sub(timeNow)):
			err = fmt.Errorf("error awaiting peer")
		}
	}
	err = fmt.Errorf("error awaiting peer")

	return
}

func (c *Connection) openStream(client *udps.Client, streamInfo *model.P2PRelayOpenStream) (stream *udp.Stream, peerStream *udp.Stream, err error) {
	streamInfo.Metadata[p2p.KEY_CONNECTION_ID] = c.id

	defer func() {
		if err != nil {
			if stream != nil {
				stream.Close()
			}
			if peerStream != nil {
				peerStream.Close()
			}
		}
	}()

	peerClient, err := c.getPeerClient(client)
	if err != nil {
		return
	}

	peerStream, err = peerClient.OpenStream(streamInfo.Metadata, streamInfo.Data)
	if err != nil {
		return
	}

	streamInfo.Metadata[p2p.KEY_STREAM_IGNORE] = "true"
	stream, err = client.OpenStream(streamInfo.Metadata, streamInfo.Data)
	if err != nil {
		return
	}

	go func() {
		defer stream.Close()
		io.Copy(stream.Stream(), peerStream.Stream())
	}()

	go func() {
		defer peerStream.Close()
		io.Copy(peerStream.Stream(), stream.Stream())
	}()

	return
}

func (c *Connection) stopTicker() {
	c.tickerTimer.Stop()
	c.ticker.Stop()
}

// Close Connection
func (c *Connection) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.Closed {
		return
	}

	c.peerConnBus.Close()
	c.server.conns.Delete(c.id)

	if client, err := c.server.udpServer.GetClient(c.getClientId(c.sourcePeer.Id)); err == nil {
		client.Close(0, "Connection Closed")
	}

	if client, err := c.server.udpServer.GetClient(c.getClientId(c.targetPeer.Id)); err == nil {
		client.Close(0, "Connection Closed")
	}

	c.Closed = true
}

// Stringify
func (c *Connection) String() string {
	return fmt.Sprintf("id(%s) peers[%s, %s] closed(%v)", c.id, c.sourcePeer.Id, c.targetPeer.Id, c.Closed)
}
