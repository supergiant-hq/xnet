package p2ps

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/p2p"
	udps "github.com/supergiant-hq/xnet/udp/server"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ConnectionIDGenerator func() (id string, err error)

// P2P Server Manager
type Manager struct {
	server          *udps.Server
	connIDGenerator ConnectionIDGenerator
	conns           *sync.Map
	rnd             *rand.Rand
	log             *logrus.Entry
}

// Create a Manager
func New(log *logrus.Logger, server *udps.Server) (m *Manager, err error) {
	m = &Manager{
		server: server,
		conns:  new(sync.Map),
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
		log:    log.WithField("prefix", "P2P"),
	}

	m.log.Info("Registering P2P Manager...")

	m.registerHandlers()

	m.log.Info("P2P Manager Registered")
	return
}

func (m *Manager) registerHandlers() {
	m.server.RegisterHandler(model.MessageTypeP2PConnectionRequest, m.connectionRequestHandler)
	m.server.RegisterHandler(model.MessageTypeP2PConnectionStatus, m.connectionStatusHandler)
	m.server.RegisterHandler(model.MessageTypeP2PRelayServers, m.getRelaysHandler)
	m.server.RegisterHandler(model.MessageTypeP2PRelayValidate, m.relayValidationHandler)
}

// Set Connection ID Generator Handler
func (m *Manager) SetConnectionIdGenerator(handler ConnectionIDGenerator) {
	m.connIDGenerator = handler
}

func (m *Manager) getPeer(id string) (p *peer, err error) {
	c, err := m.server.GetClient(id)
	if err != nil {
		return
	}

	p = newPeer(c)

	return
}

func (m *Manager) generateConnectionID() (id string, err error) {
	if m.connIDGenerator != nil {
		return m.connIDGenerator()
	}
	return uuid.NewString(), nil
}

func (m *Manager) getRelayServers() []*udps.Client {
	return m.server.GetClientsWithTag(p2p.TAG_RELAY)
}

// Close Connection
func (m *Manager) CloseConnection(cid string) {
	if rconn, ok := m.conns.LoadAndDelete(cid); ok {
		conn := rconn.(*Connection)
		conn.Close(fmt.Sprintf("Connection Closed (%s)", conn.id))
	}
}

// Close all Connections
func (m *Manager) CloseAllConnections() {
	m.log.Warnln("Closing all connections...")

	m.conns.Range(func(k, v interface{}) bool {
		conn := v.(*Connection)
		conn.Close("Close All")
		return true
	})

	m.log.Warnln("Closed all connections")
}
