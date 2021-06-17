package udps

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/udp"
	"github.com/supergiant-hq/xnet/util"

	"github.com/google/uuid"
	"github.com/lucas-clemente/quic-go"
	"github.com/sirupsen/logrus"
)

const (
	tickerInterval time.Duration = time.Second * 20
)

// Client
type Client struct {
	server *Server
	// Metadata
	Meta        *model.ClientData
	initialized bool

	sessionId string
	session   quic.Session
	channel   *network.Channel
	streams   *sync.Map

	// ID
	Id string
	// Remote Address
	Addr *net.UDPAddr
	// Tags
	Tags           map[string]string
	messageHandler map[network.MessageType]MessageHandler

	tickerTimer *time.Timer
	ticker      *util.Ticker
	exit        chan bool
	// Closed Status
	Closed bool
	log    *logrus.Entry
}

func (s *Server) newClient(log *logrus.Logger, session quic.Session) (client *Client, err error) {
	stream, err := session.AcceptStream(context.Background())
	if err != nil {
		s.log.Errorln(err)
		return
	}

	client = &Client{
		server:    s,
		Id:        "",
		Addr:      session.RemoteAddr().(*net.UDPAddr),
		sessionId: uuid.NewString(),

		session:        session,
		channel:        network.NewChannel(s.log.Logger, stream, s.Cfg.Unmarshalers()),
		streams:        new(sync.Map),
		messageHandler: make(map[network.MessageType]MessageHandler),

		exit: make(chan bool, 1),
		log:  log.WithField("prefix", fmt.Sprintf("UDPC-%s-%s", s.Cfg.Tag, session.RemoteAddr().(*net.UDPAddr).String())),
	}

	client.ticker = util.NewTicker(tickerInterval, client.handleTick)
	client.tickerTimer = time.AfterFunc(tickerInterval, client.ticker.Start)

	return
}

// Register Client level MessageHandler
func (c *Client) RegisterHandler(mtype network.MessageType, handler MessageHandler) (err error) {
	if _, ok := c.messageHandler[mtype]; ok {
		err = fmt.Errorf("handler with message type (%v) already exists", mtype)
		return
	}

	c.messageHandler[mtype] = handler

	return
}

func (c *Client) initialize(data *model.ClientData) {
	c.Id = data.Id
	c.Tags = data.Tags
	c.Meta = data
	c.initialized = true
}

// Client ID
func (c *Client) ID() string {
	return c.Id
}

// Send a Message to Client
func (c *Client) Send(msg *network.Message) (rmsg *network.Message, err error) {
	if c.channel == nil {
		err = udp.ErrorNotConnected
		return
	}
	return c.channel.Send(msg)
}

// Close Client
func (c *Client) Close(code int, reason string) {
	if c.Closed {
		return
	}

	c.CloseAllStreams()

	if c.channel != nil {
		c.channel.Close()
		c.channel = nil
	}

	c.session.CloseWithError(quic.ApplicationErrorCode(code), reason)
	c.tickerTimer.Stop()
	c.ticker.Stop()
	c.messageHandler = make(map[network.MessageType]MessageHandler)

	select {
	case c.exit <- true:
	default:
	}
	c.Closed = true

	if ec, ok := c.server.clients.Load(c.Id); ok && ec.(*Client).sessionId == c.sessionId {
		c.server.clients.Delete(c.Id)
	}
	if c.server.clientDisconnectedHandler != nil {
		go c.server.clientDisconnectedHandler(c)
	}

	c.log.Warnf("Client connection closed: %s\n", reason)
}

// Stringify
func (c *Client) String() string {
	return fmt.Sprintf("id(%s) with addr(%s)", c.Id, c.Addr.String())
}
