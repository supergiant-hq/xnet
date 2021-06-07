package p2ps

import (
	"fmt"
	"sync"
	"time"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/p2p"
	"github.com/supergiant-hq/xnet/util"

	"github.com/sirupsen/logrus"
)

// P2P Connection
type Connection struct {
	mgr *Manager

	id         string
	sourcePeer *peer
	targetPeer *peer
	mode       p2p.ConnectionMode

	tickerTimer *time.Timer
	ticker      *util.Ticker
	mutex       sync.Mutex
	// Exit Channel
	Exit chan bool
	// Closed Status
	Closed bool
	log    *logrus.Entry
}

func newConnection(log *logrus.Logger, mgr *Manager, mode p2p.ConnectionMode, sp *peer, tp *peer) (c *Connection, err error) {
	id, err := mgr.generateConnectionID()
	if err != nil {
		return
	}

	c = &Connection{
		mgr: mgr,

		id:         id,
		sourcePeer: sp,
		targetPeer: tp,
		mode:       mode,

		Exit: make(chan bool, 1),
		log:  log.WithField("prefix", "CONN"),
	}
	c.ticker = util.NewTicker(p2p.TickerDuration, c.handleTick)
	c.tickerTimer = time.AfterFunc(p2p.TickerDuration, c.ticker.Start)

	return
}

func (c *Connection) handleTick() {
	if !c.Closed {
		// Check if clients are closed
		if c.sourcePeer.client.Closed {
			c.log.Errorf(fmt.Sprintf("Peer disconnected: %s", c.sourcePeer.id))
			c.mgr.CloseConnection(c.id)
			return
		} else if c.targetPeer.client.Closed {
			c.log.Errorf(fmt.Sprintf("Peer disconnected: %s", c.targetPeer.id))
			c.mgr.CloseConnection(c.id)
			return
		}
		// Check Status
		if err := c.checkPeerStatus(c.sourcePeer); err != nil {
			c.log.Errorf(fmt.Sprintf("Peer status error: %s", err.Error()))
			c.mgr.CloseConnection(c.id)
			return
		} else if err := c.checkPeerStatus(c.targetPeer); err != nil {
			c.log.Errorf(fmt.Sprintf("Peer status error: %s", err.Error()))
			c.mgr.CloseConnection(c.id)
			return
		}
	}
}

func (c *Connection) checkPeerStatus(p *peer) (err error) {
	msg := network.NewMessageWithAck(
		model.MessageTypeP2PConnectionStatus,
		&model.P2PConnectionStatus{
			Id:     c.id,
			Status: true,
		},
		p2p.RequestTimeout,
	)
	mres, err := p.client.Send(msg)
	if err != nil {
		return
	}

	data := mres.Body.(*model.P2PConnectionStatus)
	if !data.Status {
		err = fmt.Errorf(data.Message)
		return
	}

	return
}

// Close Connection
func (c *Connection) Close(reason string) {
	if c.Closed {
		return
	}

	c.tickerTimer.Stop()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	select {
	case c.Exit <- true:
	default:
	}
	c.Closed = true

	c.log.Warnln("Connection Closed:", c.String(), reason)
}

// Stringify
func (c *Connection) String() string {
	return fmt.Sprintf("id(%s) with mode(%v) source(%v) target(%v) closed(%v)", c.id, c.mode, c.sourcePeer.id, c.targetPeer.id, c.Closed)
}
