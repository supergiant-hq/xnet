package relay

import (
	"sync"

	"github.com/supergiant-hq/xnet/model"
	udpc "github.com/supergiant-hq/xnet/udp/client"
	udps "github.com/supergiant-hq/xnet/udp/server"
	"github.com/supergiant-hq/xnet/util"

	"github.com/sirupsen/logrus"
)

// Relay Server
type Server struct {
	config    Config
	udpClient *udpc.Client
	udpServer *udps.Server

	conns      sync.Map
	connsMutex sync.Mutex

	// Open Status
	Open bool
	// Exit Channel
	Exit chan bool
	// Closed Status
	Closed bool
	log    *logrus.Logger
}

// Create a Relay Server
func New(config Config) (s *Server, err error) {
	logLevel := logrus.InfoLevel
	if config.Debug {
		logLevel = logrus.DebugLevel
	}

	if err = config.init(); err != nil {
		return
	}

	s = &Server{
		config: config,
		Exit:   make(chan bool, 1),
		log:    util.NewLogger(logLevel),
	}

	if s.udpClient, err = udpc.New(s.log, config.udpcConfig); err != nil {
		return
	}

	if s.udpServer, err = udps.New(s.log, config.udpsConfig, s.clientValidateHandler); err != nil {
		return
	}

	s.registerHandlers()

	return
}

func (s *Server) registerHandlers() {
	s.udpServer.RegisterHandler(model.MessageTypeP2PRelayOpenStream, s.openPeerStreamHandler)
	s.udpServer.RegisterHandler(model.MessageTypeP2PRelayAwait, s.awaitPeerConnection)
}

// Listen for connections
func (s *Server) Listen() (err error) {
	if s.Open {
		return
	}

	if err = s.udpServer.Listen(); err != nil {
		return
	}

	go s.udpClient.Connect()

	go func() {
		select {
		case <-s.udpClient.Exit:
		case <-s.udpServer.Exit:
		}
		s.Close()
	}()

	s.Open = true

	return
}

// Close Server
func (s *Server) Close() {
	if !s.Open || s.Closed {
		return
	}

	s.udpClient.Close(0, "Relay Server Shutdown")
	s.udpServer.Close(0, "Relay Server Shutdown")

	select {
	case s.Exit <- true:
	default:
	}
	s.Open = false
	s.Closed = true
}
