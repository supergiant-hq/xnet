package udps

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/udp"

	"github.com/lucas-clemente/quic-go"
	"github.com/sirupsen/logrus"
)

// Called on initial client connection
type ClientConnectedHandler func(*Client)

// Called when a new client connects to the server
type ClientValidateHandler func(*net.UDPAddr, *model.ClientValidateData) (cd *model.ClientData, err error)

// Called when a client disconnects
type ClientDisconnectedHandler func(*Client)

// Called when a client sends a message
type MessageHandler func(*Client, *network.Message)

const (
	KEY_CLIENT_ID = "__CLIENT_ID"
)

// Server
type Server struct {
	// Config
	Cfg Config
	// Clients Map
	clients *sync.Map

	clientConnectedHandler ClientConnectedHandler
	// Client Validation Handler
	ClientValidateHandler     ClientValidateHandler
	clientDisconnectedHandler ClientDisconnectedHandler
	// New Stream Handler
	StreamHandler  udp.StreamHandler
	messageHandler map[network.MessageType]MessageHandler

	// UDP Connection
	UDPConn  *net.UDPConn
	listener quic.Listener

	// Exit Channel
	Exit chan bool
	// Server Closed Status
	Closed bool
	mutex  sync.Mutex
	log    *logrus.Entry
}

// Creates a Server
func New(
	log *logrus.Logger,
	cfg Config,
	clientValidateHandler ClientValidateHandler,
) (s *Server, err error) {
	if err = cfg.init(false); err != nil {
		return
	}

	s = &Server{
		Cfg:     cfg,
		clients: new(sync.Map),

		ClientValidateHandler: clientValidateHandler,
		messageHandler:        make(map[network.MessageType]MessageHandler),

		Exit: make(chan bool, 1),
		log:  log.WithField("prefix", fmt.Sprintf("UDPS-%s", cfg.Tag)),
	}
	return
}

// Creates a Server using an existing net.UDPConn connection
func NewWithConnection(
	log *logrus.Logger,
	cfg Config,
	udpConn *net.UDPConn,
	clientValidateHandler ClientValidateHandler,
) (s *Server, err error) {
	if err = cfg.init(true); err != nil {
		return
	}

	s = &Server{
		Cfg:     cfg,
		clients: new(sync.Map),

		ClientValidateHandler: clientValidateHandler,
		messageHandler:        make(map[network.MessageType]MessageHandler),

		UDPConn: udpConn,
		Exit:    make(chan bool, 1),
		log:     log.WithField("prefix", fmt.Sprintf("UDPS-%s", cfg.Tag)),
	}
	return
}

// Start listening for connections
func (s *Server) Listen() (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Closed {
		return fmt.Errorf("server closed")
	}

	if s.UDPConn == nil {
		if s.UDPConn, err = net.ListenUDP("udp", s.Cfg.Addr); err != nil {
			return err
		}
	}

	if s.listener, err = quic.Listen(s.UDPConn, s.Cfg.TLS.Clone(), s.Cfg.Quic.Clone()); err != nil {
		return err
	}

	go s.handleSessions()

	s.log.Infof("Server started: %v", s.Cfg.Addr.String())

	return
}

// Set Client Connected handler
func (s *Server) SetClientConnectedHandler(handler ClientConnectedHandler) {
	s.clientConnectedHandler = handler
}

// Set Client Disconnect Handler
func (s *Server) SetClientDisconnectedHandler(handler ClientDisconnectedHandler) {
	s.clientDisconnectedHandler = handler
}

// Set New Stream Handler
func (s *Server) SetStreamHandler(handler udp.StreamHandler) {
	s.StreamHandler = handler
}

// Register a MessageType Handler
func (s *Server) RegisterHandler(mtype network.MessageType, handler MessageHandler) (err error) {
	if _, ok := s.messageHandler[mtype]; ok {
		err = fmt.Errorf("handler for message type (%v) already exists", mtype)
		return
	}

	s.messageHandler[mtype] = handler

	return
}

func (s *Server) handleSessions() {
	for {
		session, err := s.listener.Accept(context.Background())
		if err != nil {
			s.log.Errorln(err)
			return
		}

		client, err := s.newClient(s.log.Logger, session)
		if err != nil {
			s.log.Errorln(err)
			return
		}

		go func() {
			<-client.exit
		}()

		go client.handleMessages(s)

		s.log.Infoln("Client connection established:", client.Addr.String())
	}
}

func (s *Server) validateClient(c *Client, data *model.ClientValidateData) (cdata *model.ClientData, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	cdata, err = s.ClientValidateHandler(c.Addr, data)
	if err != nil {
		return
	}
	c.initialize(cdata)

	if oldrc, ok := s.clients.Load(c.Id); ok {
		oldc := oldrc.(*Client)
		s.log.Warnln("Closing old connection:", oldc.String())
		oldc.Close(400, "New client connected")
	}
	s.clients.Store(c.Id, c)

	return
}

// Get connected Client by ID
func (s *Server) GetClient(id string) (c *Client, err error) {
	if rc, ok := s.clients.Load(id); !ok {
		err = fmt.Errorf("client not found: %s", id)
	} else {
		c = rc.(*Client)
	}
	return
}

// Get Clients by Tag
func (s *Server) GetClientsWithTag(tag string) (clients []*Client) {
	clients = []*Client{}
	s.clients.Range(func(key, value interface{}) bool {
		client := value.(*Client)
		if _, ok := client.Tags[tag]; ok {
			clients = append(clients, client)
		}
		return true
	})
	return
}

// Close Server
func (s *Server) Close(code int, reason string) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Closed {
		return fmt.Errorf("server closed")
	}

	s.clients.Range(func(k, v interface{}) bool {
		c := v.(*Client)
		c.Close(code, reason)
		s.clients.Delete(k)
		return true
	})

	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}

	if !s.Cfg.managed {
		s.UDPConn.Close()
	}

	select {
	case s.Exit <- true:
	default:
	}
	s.Closed = true

	s.log.Warnln("Server shutdown complete")
	return
}
