package brokerc

import (
	"github.com/supergiant-hq/xnet/p2p"
	p2pc "github.com/supergiant-hq/xnet/p2p/client"
	"github.com/supergiant-hq/xnet/udp"
	udpc "github.com/supergiant-hq/xnet/udp/client"
	"github.com/supergiant-hq/xnet/util"

	"github.com/sirupsen/logrus"
)

// Broker Client
type Client struct {
	config     Config
	udpClient  *udpc.Client
	p2pManager *p2pc.Manager

	// Connect to peer by ID
	ConnectPeerById func(peerId string, mode p2p.ConnectionMode) (conn *p2pc.Connection, err error)
	// Connect to peer by Tag
	ConnectPeerByTag func(tag string, mode p2p.ConnectionMode) (conn *p2pc.Connection, err error)

	// Client exit channel
	Exit chan bool
	log  *logrus.Logger
}

// Create Client
func New(
	config Config,
	streamHandler udp.StreamHandler,
) (c *Client, err error) {
	logLevel := logrus.InfoLevel
	if config.Debug {
		logLevel = logrus.DebugLevel
	}

	c = &Client{
		config: config,
		log:    util.NewLogger(logLevel),
	}

	c.udpClient, err = udpc.New(c.log, config.UdpcConfig)
	if err != nil {
		return
	}

	if c.p2pManager, err = p2pc.New(c.log, config.P2PConfig, c.udpClient); err != nil {
		return
	}
	c.p2pManager.SetStreamHandler(streamHandler)

	c.ConnectPeerById = c.p2pManager.ConnectById
	c.ConnectPeerByTag = c.p2pManager.ConnectByTag

	return
}

// Set New Connection Handler
func (c *Client) SetConnectionHandler(handler p2pc.ConnectionHandler) {
	c.p2pManager.SetConnectionHandler(handler)
}

// Connect to Broker Server
func (c *Client) Connect() (err error) {
	if err = c.udpClient.Connect(); err != nil {
		return
	}

	go func() {
		<-c.udpClient.Exit
		c.Close()
	}()

	return
}

// Close Client
func (c *Client) Close() {
	c.p2pManager.CloseAll()
	c.udpClient.Close(0, "Client Shutdown")
	select {
	case c.Exit <- true:
	default:
	}
}
