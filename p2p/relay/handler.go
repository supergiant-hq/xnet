package relay

import (
	"fmt"
	"net"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/udp"
	udps "github.com/supergiant-hq/xnet/udp/server"
)

func (rs *Server) clientValidateHandler(addr *net.UDPAddr, data *model.ClientValidateData) (cd *model.ClientData, err error) {
	rs.connsMutex.Lock()
	defer rs.connsMutex.Unlock()

	connData, err := rs.validateConnection(data)
	if err != nil {
		rs.log.Errorln("Error accepting client: ", err.Error())
		return
	}

	var conn *Connection
	rconn, ok := rs.conns.Load(connData.Id)
	if ok {
		conn = rconn.(*Connection)
	} else {
		conn = rs.NewConnection(connData)
	}

	cd = &model.ClientData{
		Id:      conn.getClientId(connData.Peer.Id),
		Address: addr.String(),
		Tag:     connData.Id,
		Data:    data.Data,
		Ctx: &model.ClientData_RelayCtx{
			RelayCtx: &model.RelayClientContext{
				ConnectionId: connData.Id,
				SourcePeerId: connData.SourcePeer.Id,
				TargetPeerId: connData.TargetPeer.Id,
			},
		},
	}

	conn.peerConnBus.TryPub(connData.Peer.Id)

	return
}

func (rs *Server) validateConnection(data *model.ClientValidateData) (relayData *model.P2PRelayConnectionData, err error) {
	msg := network.NewMessageWithAck(
		model.MessageTypeP2PRelayValidate,
		&model.ClientValidateData{
			Token: data.Token,
			Data:  data.Data,
		},
		network.RequestTimeout,
	)
	rmsg, err := rs.udpClient.Send(msg)
	if err != nil {
		return
	}

	relayData = rmsg.Body.(*model.P2PRelayConnectionData)
	if !relayData.Status {
		err = fmt.Errorf(relayData.Message)
		return
	}

	return
}

func (s *Server) getConnection(connId string) (conn *Connection, err error) {
	rconn, ok := s.conns.Load(connId)
	if !ok {
		err = fmt.Errorf("connection not found")
		return
	}
	conn = rconn.(*Connection)

	_, err = s.udpServer.GetClient(conn.getClientId(conn.sourcePeer.Id))
	if err != nil {
		return
	}

	_, err = s.udpServer.GetClient(conn.getClientId(conn.targetPeer.Id))
	if err != nil {
		return
	}

	return
}

func (s *Server) awaitPeerConnection(c *udps.Client, msg *network.Message) {
	var err error

	s.log.Infof("Await peer connection by (%s)", c.Id)

	defer func() {
		var rmsgData model.P2PRelayStreamInfo

		if err != nil {
			s.log.Errorf("Error awaiting peer connection by (%s): %v", c.Id, err.Error())
			rmsgData = model.P2PRelayStreamInfo{
				Status:  false,
				Message: err.Error(),
			}
		} else {
			s.log.Infoln("Peer connected to relay")
			rmsgData = model.P2PRelayStreamInfo{
				Status:  true,
				Message: "Ok",
			}
		}
		rmsg, _ := msg.GenReply(model.MessageTypeP2PRelayStreamInfo, &rmsgData)
		c.Send(rmsg)
	}()

	connId := c.Meta.GetRelayCtx().ConnectionId
	conn, err := s.getConnection(connId)
	if err != nil {
		return
	}

	err = conn.awaitPeer(c)
}

func (s *Server) openPeerStreamHandler(c *udps.Client, msg *network.Message) {
	var err error
	var stream *udp.Stream

	s.log.Infof("Open peer connection request from (%s)", c.Id)

	defer func() {
		var rmsgData model.P2PRelayStreamInfo

		if err != nil {
			s.log.Errorf("Error opening peer connection from (%s): %v", c.Id, err.Error())
			rmsgData = model.P2PRelayStreamInfo{
				Status:  false,
				Message: err.Error(),
			}
		} else {
			s.log.Infof("Opened peer connection from (%s)", c.Id)
			rmsgData = model.P2PRelayStreamInfo{
				Status:  true,
				Message: "Ok",
				Id:      stream.Id,
			}
		}
		rmsg, _ := msg.GenReply(model.MessageTypeP2PRelayStreamInfo, &rmsgData)
		c.Send(rmsg)
	}()

	msgData := msg.Body.(*model.P2PRelayOpenStream)

	connId := c.Meta.GetRelayCtx().ConnectionId
	conn, err := s.getConnection(connId)
	if err != nil {
		return
	}

	if stream, _, err = conn.openStream(c, msgData.Data); err != nil {
		return
	}
}
