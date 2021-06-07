package brokers

import (
	p2ps "github.com/supergiant-hq/xnet/p2p/server"
	udps "github.com/supergiant-hq/xnet/udp/server"
	"github.com/supergiant-hq/xnet/util"

	"github.com/sirupsen/logrus"
)

// Broker Server
type Server struct {
	config     Config
	udpServer  *udps.Server
	p2pManager *p2ps.Manager

	// Server Open
	Open bool
	// Exit Channel
	Exit chan bool
	// Closed Status
	Closed bool
	log    *logrus.Logger
}

// New Broker Server
func New(config Config, cvh udps.ClientValidateHandler) (s *Server, err error) {
	logLevel := logrus.InfoLevel
	if config.Debug {
		logLevel = logrus.DebugLevel
	}

	s = &Server{
		config: config,
		Exit:   make(chan bool, 1),
		log:    util.NewLogger(logLevel),
	}

	s.udpServer, err = udps.New(s.log, config.UdpsConfig, cvh)
	if err != nil {
		return
	}

	s.p2pManager, err = p2ps.New(s.log, s.udpServer)
	if err != nil {
		return
	}

	return
}

// Listen for connections
func (s *Server) Listen() (err error) {
	if s.Open {
		return
	}

	if err = s.udpServer.Listen(); err != nil {
		return
	}

	go func() {
		<-s.udpServer.Exit
		s.Close()
	}()

	s.Open = true

	return
}

// Close Broker Server
func (s *Server) Close() {
	if !s.Open || s.Closed {
		return
	}

	s.p2pManager.CloseAllConnections()
	s.udpServer.Close(0, "Broker Server Shutdown")

	select {
	case s.Exit <- true:
	default:
	}
	s.Open = false
	s.Closed = true
}
