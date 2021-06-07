package p2ps

import (
	"fmt"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/p2p"
	udps "github.com/supergiant-hq/xnet/udp/server"
	"github.com/supergiant-hq/xnet/util"
)

func (m *Manager) connectionRequestHandler(c *udps.Client, msg *network.Message) {
	var err error
	var pd *model.P2PPeerData
	var conn *Connection

	connData := msg.Body.(*model.P2PConnectionRequest)

	defer func() {
		var resMsg model.P2PConnectionData

		if err != nil {
			m.log.Errorln("Error creating connection:", err.Error())

			resMsg = model.P2PConnectionData{
				Status:  false,
				Message: err.Error(),
			}
		} else {
			resMsg = model.P2PConnectionData{
				Status:       true,
				Message:      "Created",
				Id:           conn.id,
				Mode:         string(conn.mode),
				RelayAddress: connData.RelayAddress,
				Peer: &model.P2PPeerData{
					Id:        conn.targetPeer.id,
					Address:   conn.targetPeer.client.Addr.String(),
					Addresses: util.RemoveDuplicatesFromSlice(append(pd.Addresses, pd.Address, conn.targetPeer.client.Addr.String())),
				},
			}

			m.conns.Store(conn.id, conn)
		}

		rmsg, _ := msg.GenReply(model.MessageTypeP2PConnectionData, &resMsg)
		c.Send(rmsg)
	}()

	clientBrokerCtx := c.Meta.GetBrokerCtx()
	if clientBrokerCtx == nil {
		err = fmt.Errorf("client does not have a broker context")
		return
	}

	conn, pd, err = m.createConnection(c.Id, connData)
	if err != nil {
		return
	}
}

func (m *Manager) createConnection(sourceId string, req *model.P2PConnectionRequest) (c *Connection, pd *model.P2PPeerData, err error) {
	defer func() {
		if err != nil && c != nil {
			m.CloseConnection(c.id)
		}
	}()

	mode := p2p.ConnectionMode(req.Mode)

	m.log.Infof("Connection request from(%s) to(%s) via(%s)", sourceId, req.Peer.Id, mode)

	if mode == p2p.ConnectionModeRelay && len(req.RelayAddress) == 0 {
		err = fmt.Errorf("relay address must be provided")
		return
	}

	sp, err := m.getPeer(sourceId)
	if err != nil {
		return
	}
	tp, err := m.getPeer(req.Peer.Id)
	if err != nil {
		return
	}

	c, err = newConnection(m.log.Logger, m, mode, sp, tp)
	if err != nil {
		return
	}

	// Get confirmation from TargetPeer
	msg := network.NewMessageWithAck(
		model.MessageTypeP2PConnectionRequest,
		&model.P2PConnectionRequest{
			Id:           c.id,
			Mode:         string(c.mode),
			RelayAddress: req.RelayAddress,
			Peer: &model.P2PPeerData{
				Id:        c.sourcePeer.id,
				Address:   c.sourcePeer.client.Addr.String(),
				Addresses: util.RemoveDuplicatesFromSlice(append(req.Peer.Addresses, req.Peer.Address, c.sourcePeer.client.Addr.String())),
			},
		},
		p2p.RequestTimeout,
	)
	rmsg, err := tp.client.Send(msg)
	if err != nil {
		return
	}
	rcd := rmsg.Body.(*model.P2PConnectionData)
	if !rcd.Status {
		err = fmt.Errorf(rcd.Message)
		return
	}
	pd = rcd.Peer

	m.log.Infoln("Connection initialization complete:", c.String())
	return
}

func (m *Manager) connectionStatusHandler(c *udps.Client, msg *network.Message) {
	var err error

	defer func() {
		rdata := model.P2PConnectionStatus{}

		if err != nil {
			rdata.Status = false
			rdata.Message = err.Error()
		} else {
			rdata.Status = true
			rdata.Message = "Ok"
		}

		rmsg, _ := msg.GenReply(model.MessageTypeP2PConnectionStatus, &rdata)
		c.Send(rmsg)
	}()

	reqData := msg.Body.(*model.P2PConnectionStatus)

	rconn, ok := m.conns.Load(reqData.Id)
	if !ok {
		err = fmt.Errorf("connection not found")
		return
	}
	conn := rconn.(*Connection)
	if conn.Closed {
		err = fmt.Errorf("connection closed")
		return
	}
}

func (m *Manager) getRelaysHandler(c *udps.Client, msg *network.Message) {
	relays := m.getRelayServers()

	servers := []string{}
	for _, relay := range relays {
		port, ok := relay.Meta.Data[p2p.KEY_PORT]
		if !ok {
			m.log.Warnln("Relay did not send the PORT key in DATA")
			continue
		}
		servers = append(servers, fmt.Sprintf("%s:%s", relay.Addr.IP.String(), port))
	}
	rmsg, err := msg.GenReply(model.MessageTypeP2PRelayServers, &model.P2PRelayServers{
		Servers: servers,
	})
	if err != nil {
		return
	}
	c.Send(rmsg)
}

func (m *Manager) relayValidationHandler(c *udps.Client, msg *network.Message) {
	var err error
	var peer model.P2PPeerData
	var conn *Connection

	defer func() {
		var resMsg model.P2PRelayConnectionData

		if err != nil {
			resMsg = model.P2PRelayConnectionData{
				Status:  false,
				Message: err.Error(),
			}
		} else {
			resMsg = model.P2PRelayConnectionData{
				Status:  !conn.Closed,
				Message: "OK",

				Id:   conn.id,
				Mode: string(conn.mode),
				Peer: &peer,
				SourcePeer: &model.P2PPeerData{
					Id: conn.sourcePeer.id,
				},
				TargetPeer: &model.P2PPeerData{
					Id: conn.targetPeer.id,
				},
			}
		}

		rmsg, _ := msg.GenReply(model.MessageTypeP2PRelayConnectionData, &resMsg)
		c.Send(rmsg)
	}()

	reqData := msg.Body.(*model.ClientValidateData)
	connId := reqData.Data[p2p.KEY_CONNECTION_ID]

	clientData, err := m.server.ClientValidateHandler(c.Addr, reqData)
	if err != nil {
		return
	}

	rconn, ok := m.conns.Load(connId)
	if !ok {
		err = fmt.Errorf("connection not found")
		return
	}
	conn = rconn.(*Connection)
	if conn.sourcePeer.id != clientData.Id && conn.targetPeer.id != clientData.Id {
		err = fmt.Errorf("invalid connection id")
		return
	}

	peer = model.P2PPeerData{
		Id: clientData.Id,
	}
}
