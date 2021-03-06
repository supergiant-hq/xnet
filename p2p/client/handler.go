package p2pc

import (
	"fmt"
	"net"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/p2p"
	"github.com/supergiant-hq/xnet/tun"
	"github.com/supergiant-hq/xnet/udp"
	udpc "github.com/supergiant-hq/xnet/udp/client"
	udps "github.com/supergiant-hq/xnet/udp/server"
)

// Client
func (m *Manager) clientDisconnectedHandler(c *udps.Client) {
	ctx := c.Meta.GetP2PCtx()
	if ctx == nil {
		return
	}

	if ctx.Active {
		m.CloseConnection(ctx.ConnId, "Exited")
	}
}

// Connection
func (m *Manager) connectionRequestHandler(c *udpc.Client, msg *network.Message) {
	var err error
	var conn *Connection
	var interfaces []string

	defer func() {
		var resData model.P2PConnectionData

		if err != nil {
			m.log.Errorln("Error accepting connection:", err.Error())

			resData = model.P2PConnectionData{
				Status:  false,
				Message: err.Error(),
			}
		} else {
			resData = model.P2PConnectionData{
				Status:  true,
				Message: "Ok",
				Peer: &model.P2PPeerData{
					Id:        c.Id,
					Address:   c.Addr.String(),
					Addresses: interfaces,
				},
			}
		}

		rmsg, err := msg.GenReply(
			model.MessageTypeP2PConnectionData,
			&resData,
		)
		if err != nil {
			m.log.Errorln(err)
			return
		}

		c.Send(rmsg)
	}()

	creq := msg.Body.(*model.P2PConnectionRequest)

	m.log.Infof("Connection request from peer id(%s) ip(%s)", creq.Peer.Id, creq.Peer.Address)

	interfaceIPs, err := tun.GetInterfaceAddresses()
	if err != nil {
		return
	}
	for _, ip := range interfaceIPs {
		interfaces = append(interfaces, fmt.Sprintf("%s:%d", ip.String(), c.Addr.Port))
	}

	if conn, err = acceptConnection(m.log.Logger, m, creq); err != nil {
		return
	}
	m.conns.Store(conn.id, conn)

	conn.log.Infof("Created connection: %s", conn.String())
}

func (m *Manager) connectionStatusHandler(c *udpc.Client, msg *network.Message) {
	var err error

	defer func() {
		rdata := new(model.P2PConnectionStatus)

		if err != nil {
			rdata.Status = false
			rdata.Message = err.Error()
		} else {
			rdata.Status = true
			rdata.Message = "Ok"
		}

		rmsg, _ := msg.GenReply(model.MessageTypeP2PConnectionStatus, rdata)
		c.Send(rmsg)
	}()

	mdata := msg.Body.(*model.P2PConnectionStatus)
	rconn, ok := m.conns.Load(mdata.Id)
	if !ok {
		err = fmt.Errorf("connection not found")
		return
	}
	conn := rconn.(*Connection)

	if !conn.IsConnected() {
		err = fmt.Errorf("connection pending")
		return
	}
}

func (m *Manager) clientValidateHandler(addr *net.UDPAddr, data *model.ClientValidateData) (cdata *model.ClientData, err error) {
	rconn, ok := m.conns.Load(data.Token)
	if !ok {
		err = fmt.Errorf("connection with id (%s) not found", data.Token)
		return
	}
	conn := rconn.(*Connection)

	if conn.mode != p2p.ConnectionModeP2P {
		err = fmt.Errorf("connection not in p2p mode")
		return
	} else if conn.p2pConn == nil {
		err = fmt.Errorf("peer not ready")
		return
	}

	validIP := false
	for _, paddr := range conn.peer.addrs {
		if err != nil {
			continue
		} else if paddr.IP.String() == addr.IP.String() {
			validIP = true
			break
		}
	}
	if !validIP {
		err = fmt.Errorf("peer address mismatch")
		return
	}

	cdata = &model.ClientData{
		Id:      fmt.Sprintf("%s:%s", conn.id, addr.String()),
		Address: addr.String(),
		Data:    data.Data,
		Ctx: &model.ClientData_P2PCtx{
			P2PCtx: &model.P2PClientContext{
				ConnId: conn.id,
				PeerId: conn.peer.id,
				Active: false,
			},
		},
	}

	return
}

func (m *Manager) clientInitHandler(c *udps.Client, msg *network.Message) {
	var err error

	defer func() {
		rdata := new(model.P2PConnectionStatus)

		if err != nil {
			rdata.Status = false
			rdata.Message = err.Error()
		} else {
			rdata.Status = true
			rdata.Message = "Ok"
		}

		rmsg, _ := msg.GenReply(model.MessageTypeP2PConnectionStatus, rdata)
		c.Send(rmsg)
	}()

	ctx := c.Meta.GetP2PCtx()
	rconn, ok := m.conns.Load(ctx.ConnId)
	if !ok {
		err = fmt.Errorf("connection with id (%s) not found", ctx.ConnId)
		return
	}
	conn := rconn.(*Connection)

	if err = conn.p2pConn.checkinRemoteClient(c); err != nil {
		return
	}
	ctx.Active = true
}

func (m *Manager) incomingStreamHandler(client udp.Client, stream *udp.Stream) {
	// Check if the stream should be ignored
	// This key is present if the stream was opened using
	// the OpenStream function of this client.
	// Thus, we do not need to pass it to the custom handler
	// as this is not really an incoming stream
	if _, ok := stream.Metadata[p2p.KEY_STREAM_IGNORE]; ok {
		return
	}

	// Stream is a message stream.
	// It's opened to exchange structured messages (network.Message)
	if _, ok := stream.Metadata[p2p.KEY_STREAM_MESSAGE]; ok {
		if m.messageStreamHandler == nil {
			m.log.Errorln("Incoming stream error: MessageStream handler not found")
			stream.Close()
			return
		}

		conn, ok := m.conns.Load(stream.Metadata[p2p.KEY_CONNECTION_ID])
		if !ok {
			m.log.Errorln("Incoming stream error: MessageStream connection not found")
			stream.Close()
			return
		}

		go m.messageStreamHandler(NewMessageStream(conn.(*Connection), stream))

		return
	}

	// Custom stream handler
	if m.streamHandler == nil {
		m.log.Errorf("StreamHandler is not found")
		return
	}
	go m.streamHandler(client, stream)
}
