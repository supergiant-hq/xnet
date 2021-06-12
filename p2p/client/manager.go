package p2pc

import (
	"fmt"
	"math/rand"
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

// Called on new Connection
type ConnectionHandler func(c *Connection)

// Called on new MessageStream
type MessageStreamHandler func(ms *MessageStream)

// P2P Client Manager
type Manager struct {
	config     Config
	peerServer *udps.Server
	client     *udpc.Client

	conns                *sync.Map
	connectionHandler    ConnectionHandler
	streamHandler        udp.StreamHandler
	messageStreamHandler MessageStreamHandler

	rnd *rand.Rand
	log *logrus.Entry
}

// Create new Manager
func New(
	log *logrus.Logger,
	config Config,
	client *udpc.Client,
) (m *Manager, err error) {
	m = &Manager{
		client: client,
		conns:  new(sync.Map),
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
		log:    log.WithField("prefix", "P2PM"),
	}
	m.log.Infoln("Registering P2P Manager...")

	if m.peerServer, err = udps.NewWithConnection(
		m.log.Logger,
		udps.Config{
			Tag:         "P2P",
			TLS:         client.Cfg.TLS.Clone(),
			Quic:        client.Cfg.Quic.Clone(),
			Unmarshaler: client.Cfg.Unmarshaler,
		},
		client.UDPConn,
		m.clientValidateHandler,
	); err != nil {
		return
	}

	if err = m.peerServer.Listen(); err != nil {
		return
	}

	m.peerServer.SetStreamHandler(m.incomingStreamHandler)

	m.registerHandlers()

	m.log.Infoln("P2P Manager Registered")
	return
}

// Set New Connection Handler
func (m *Manager) SetConnectionHandler(handler ConnectionHandler) {
	m.connectionHandler = handler
}

// Set New Stream Handler
func (m *Manager) SetStreamHandler(handler udp.StreamHandler) {
	m.streamHandler = handler
}

// Set Message Stream Handler
func (m *Manager) SetMessageStreamHandler(handler MessageStreamHandler) {
	m.messageStreamHandler = handler
}

func (m *Manager) registerHandlers() {
	m.peerServer.SetClientDisconnectedHandler(m.clientDisconnectedHandler)
	m.peerServer.RegisterHandler(model.MessageTypeP2PClientInit, m.clientInitHandler)
	m.client.RegisterHandler(model.MessageTypeP2PConnectionRequest, m.connectionRequestHandler)
	m.client.RegisterHandler(model.MessageTypeP2PConnectionStatus, m.connectionStatusHandler)
}

func (m *Manager) getNearestRelay() (addr string, err error) {
	// Hardcoded
	if m.config.RelayAddr != nil {
		return m.config.RelayAddr.String(), nil
	}

	// Dynamic
	msg := network.NewMessageWithAck(
		model.MessageTypeP2PRelayServers,
		&model.P2PRelayServers{},
		network.RequestTimeout,
	)
	rmsg, err := m.client.Send(msg)
	if err != nil {
		return
	}

	servers := rmsg.Body.(*model.P2PRelayServers).Servers
	if len(servers) == 0 {
		err = fmt.Errorf("no relay servers found")
		return
	}

	serverIPs := []string{}
	serverIPMap := map[string]string{}
	for _, raddr := range servers {
		addr, err := net.ResolveUDPAddr("udp", raddr)
		if err != nil {
			continue
		}
		serverIPs = append(serverIPs, addr.IP.String())
		serverIPMap[addr.IP.String()] = raddr
	}
	pingRes := network.PingAddrs(serverIPs, true)
	if len(pingRes.Success) == 0 {
		err = fmt.Errorf("error pinging relay servers: %+v", pingRes)
		return
	}
	addr = serverIPMap[pingRes.Success[0].Addr]

	return
}

// Connect to Client by ID
func (m *Manager) ConnectById(peerId string, mode p2p.ConnectionMode) (conn *Connection, err error) {
	m.log.Infof("Connecting to peer id(%s) using mode(%v)...", peerId, mode)

	if conn, err = createConnection(m.log.Logger, m, peerId, mode); err != nil {
		return
	}
	m.conns.Store(conn.id, conn)

	m.log.Infof("Created connection: %s", conn.String())
	return
}

// Connect to Client by Tag
func (m *Manager) ConnectByTag(tag string, mode p2p.ConnectionMode) (conn *Connection, err error) {
	m.log.Infof("Connecting to peer tag(%s) using mode(%v)...", tag, mode)

	clients, err := m.client.SearchClients(&model.ClientSearch{
		Tag: tag,
	})
	if err != nil {
		return
	}
	clientId := clients[m.rnd.Intn(len(clients))]

	if conn, err = createConnection(m.log.Logger, m, clientId, mode); err != nil {
		return
	}
	m.conns.Store(conn.id, conn)

	m.log.Infof("Created connection: %s", conn.String())
	return
}

// Close Connection
func (m *Manager) CloseConnection(cid string, reason string) {
	if c, ok := m.conns.LoadAndDelete(cid); ok {
		conn := c.(*Connection)
		conn.Close(reason)
		m.log.Warnf("Connection closed (%s)", conn.id)
	}
}

// Close All Connections
func (m *Manager) CloseAll() {
	m.log.Warnln("Closing all connections...")

	m.conns.Range(func(k, v interface{}) bool {
		m.CloseConnection(k.(string), "Closing all")
		return true
	})

	m.log.Warnln("Closed all connections")
}
