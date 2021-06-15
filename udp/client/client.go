package udpc

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/p2p"
	"github.com/supergiant-hq/xnet/udp"

	"github.com/lucas-clemente/quic-go"
	"github.com/sirupsen/logrus"
)

// Called to determine whether to connect to a Server
type CanConnect func(tries int) bool

// Called on connection to the server
type ConnectedHandler func(reconnect bool)

// Called to determine whether to reconnect to a Server
type CanReconnect func(tries int) bool

// Called on disconnection from a Server
type DisconnectedHandler func()

// Called when client is closed
type ClosedHandler func(reason string)

// Called when a Server sends a message
type MessageHandler func(*Client, *network.Message)

// Client
type Client struct {
	// Config
	Cfg                 Config
	canConnect          CanConnect
	connectedHandler    ConnectedHandler
	canReconnect        CanReconnect
	disconnectedHandler DisconnectedHandler
	closedHandler       ClosedHandler
	messageHandler      map[network.MessageType]MessageHandler
	streamHandler       udp.StreamHandler

	// Listen Address
	Addr *net.UDPAddr
	// UDP Connection
	// It's exposed as it's needed in the p2pc package
	UDPConn *net.UDPConn
	session quic.Session
	channel *network.Channel
	streams *sync.Map

	// ID
	Id        string
	sessionId string
	// Metadata
	Data *model.ClientData
	init bool

	// Connected Status
	Connected bool
	// Exit Channel
	Exit chan bool
	// Closed Status
	Closed bool
	mutex  sync.Mutex
	log    *logrus.Entry
}

// Creates a new Client
func New(log *logrus.Logger, cfg Config) (c *Client, err error) {
	if err = cfg.init(false); err != nil {
		return
	}

	c = &Client{
		Cfg:            cfg,
		messageHandler: make(map[network.MessageType]MessageHandler),
		streams:        new(sync.Map),

		Exit: make(chan bool, 1),
		log:  log.WithField("prefix", fmt.Sprintf("UDPC-%s", cfg.Tag)),
	}

	if c.Addr, err = net.ResolveUDPAddr("udp", ":0"); err != nil {
		return
	}

	if c.UDPConn, err = net.ListenUDP("udp", c.Addr); err != nil {
		return
	}

	return
}

// Creates a new Client using an existing net.UDPConn connection
func NewWithConnection(log *logrus.Logger, cfg Config, addr *net.UDPAddr, udpConn *net.UDPConn) (c *Client, err error) {
	if err = cfg.init(true); err != nil {
		return
	}

	c = &Client{
		Cfg:            cfg,
		messageHandler: make(map[network.MessageType]MessageHandler),
		streams:        new(sync.Map),

		Addr:    addr,
		UDPConn: udpConn,

		Exit: make(chan bool, 1),
		log:  log.WithField("prefix", fmt.Sprintf("UDPC-%s", cfg.Tag)),
	}

	return
}

// Set CanConnect Handler
func (c *Client) SetCanConnectHandler(handler CanConnect) {
	c.canConnect = handler
}

// Set Connected Handler
func (c *Client) SetConnectedHandler(handler ConnectedHandler) {
	c.connectedHandler = handler
}

// Set CanReconnect Handler
func (c *Client) SetCanReconnectHandler(handler CanReconnect) {
	c.canReconnect = handler
}

// Set Disconnected Handler
func (c *Client) SetDisconnectedHandler(handler DisconnectedHandler) {
	c.disconnectedHandler = handler
}

// Set Closed Handler
func (c *Client) SetClosedHandler(handler ClosedHandler) {
	c.closedHandler = handler
}

// Set New Stream Handler
func (c *Client) SetStreamHandler(handler udp.StreamHandler) {
	c.streamHandler = handler
}

// Register a MessageType Handler
func (c *Client) RegisterHandler(mtype network.MessageType, handler MessageHandler) (err error) {
	if _, ok := c.messageHandler[mtype]; ok {
		err = fmt.Errorf("handler with message type (%v) already exists", mtype)
		return
	}

	c.messageHandler[mtype] = handler

	return
}

// Connect to Server
func (c *Client) Connect() (err error) {
	err = fmt.Errorf("did not try to connect cause of canConnect handler")

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.Closed {
		return fmt.Errorf("client closed")
	}

	canConnect := c.canConnectHandler
	if c.canConnect != nil {
		canConnect = c.canConnect
	}
	for tries := 0; canConnect(tries); tries++ {
		c.log.Warnf("Connecting to (%s) try(%d)...", c.Cfg.ServerAddr.String(), tries+1)
		if err = c.connect(); err != nil {
			if tries != c.Cfg.ConnectTries {
				c.log.Warnf("Could not connect to (%s): %v", c.Cfg.ServerAddr.String(), err.Error())
				<-time.After(time.Second)
			}
			continue
		}

		break
	}
	if err != nil {
		return
	}
	c.log.Infof("Connection established with (%s)", c.Cfg.ServerAddr.String())

	if err = c.initialize(); err != nil {
		go c.Close(0, err.Error())
		return
	}

	if c.connectedHandler != nil {
		go c.connectedHandler(false)
	}

	c.log.Infof("Connected to (%s)", c.Cfg.ServerAddr.String())

	return
}

func (c *Client) connect() (err error) {
	c.reset()

	if c.session, err = quic.Dial(
		c.UDPConn,
		c.Cfg.ServerAddr,
		c.Cfg.ServerAddr.String(),
		c.Cfg.TLS.Clone(),
		c.Cfg.Quic.Clone(),
	); err != nil {
		return
	}

	stream, err := c.session.OpenStream()
	if err != nil {
		return
	}

	c.channel = network.NewChannel(c.log.Logger, stream, c.Cfg.Unmarshalers())
	c.sessionId = time.Now().UTC().String()

	return
}

func (c *Client) canConnectHandler(tries int) bool {
	return c.Cfg.ConnectTries == 0 || tries < c.Cfg.ConnectTries
}

func (c *Client) canReconnectHandler(tries int) bool {
	return c.Cfg.ReconnectTries == 0 || tries < c.Cfg.ReconnectTries
}

func (c *Client) reconnect() {
	err := fmt.Errorf("did not try to connect cause of canReconnect handler")

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.Closed {
		return
	}

	canReconnect := c.canReconnectHandler
	if c.canReconnect != nil {
		canReconnect = c.canReconnect
	}
	for tries := 0; canReconnect(tries); tries++ {
		c.log.Warnf("Reconnecting to (%s) try(%d)...", c.Cfg.ServerAddr.String(), tries+1)
		if err = c.connect(); err != nil {
			c.log.Warnf("Could not connect to (%s): %v", c.Cfg.ServerAddr.String(), err.Error())
			<-time.After(time.Second)
		} else if err = c.initialize(); err != nil {
			c.log.Warnf("Could not initialize client: %s", err.Error())
			if c.Data != nil && !c.Data.Status {
				break
			}
		} else {
			if c.connectedHandler != nil {
				go c.connectedHandler(true)
			}
			c.log.Infof("Reconnected to (%s)", c.Cfg.ServerAddr.String())
			return
		}
	}

	c.log.Errorf("Could not connect to (%s): %v", c.Cfg.ServerAddr.String(), err.Error())
	go c.Close(503, "Could not reconnect")
}

func (c *Client) initialize() (err error) {
	msg := network.NewMessageWithAck(
		model.MessageTypeClientValidate,
		&model.ClientValidateData{
			Token: c.Cfg.Token,
			Data:  c.Cfg.Data,
		},
		p2p.RequestTimeout,
	)
	mres, err := c.sendAndRead(msg)
	if err != nil {
		return
	}

	clientData := mres.Body.(*model.ClientData)
	if err = c.handleInitData(clientData); err != nil {
		return
	}

	go c.handlePings(c.sessionId)
	go c.handleMessages(c.sessionId)
	go c.handleStreams(c.sessionId)

	c.Connected = true

	return
}

// Client ID
func (c *Client) ID() string {
	return c.Id
}

// Send a Message to the Server
func (c *Client) Send(msg *network.Message) (rmsg *network.Message, err error) {
	return c.channel.Send(msg)
}

func (c *Client) sendAndRead(msg *network.Message) (rmsg *network.Message, err error) {
	return c.channel.SendAndRead(msg)
}

func (c *Client) reset() {
	c.Connected = false

	if c.channel != nil {
		c.channel.Close()
		c.channel = nil
	}

	c.CloseAllStreams()

	if c.session != nil {
		c.session.CloseWithError(quic.ApplicationErrorCode(0), "Reset")
		c.session = nil
	}
}

// Close connection with Server
func (c *Client) Close(code int, reason string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.Closed {
		return
	}

	c.reset()
	if !c.Cfg.managed {
		c.UDPConn.Close()
	}

	select {
	case c.Exit <- true:
	default:
	}
	c.Closed = true

	if c.closedHandler != nil {
		go c.closedHandler(reason)
	}

	c.log.Warnf("Connection to (%s) closed: %s\n", c.Cfg.ServerAddr.String(), reason)
}

// Stringify
func (c *Client) String() string {
	return fmt.Sprintf("id(%s) with addr(%s)", c.Id, c.Addr.String())
}
