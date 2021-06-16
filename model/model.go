// This package hosts the MessageTypes and their mappings for Unmarshaling data
package model

import (
	"errors"
	"fmt"

	"github.com/supergiant-hq/xnet/network"

	"google.golang.org/protobuf/proto"
)

const (
	MessageTypeClientPing     = network.MessageType("network-client-ping")
	MessageTypeClientValidate = network.MessageType("network-client-validate")
	MessageTypeClientData     = network.MessageType("network-client-data")
	MessageTypeClientSearch   = network.MessageType("network-client-search")
	MessageTypeClients        = network.MessageType("network-clients")

	MessageTypeStreamConnectionData   = network.MessageType("network-stream-conn-data")
	MessageTypeStreamConnectionStatus = network.MessageType("network-stream-conn-status")

	MessageTypeP2PClientInit        = network.MessageType("p2p-client-init")
	MessageTypeP2PConnectionRequest = network.MessageType("p2p-conn-request")
	MessageTypeP2PConnectionStatus  = network.MessageType("p2p-conn-status")
	MessageTypeP2PConnectionData    = network.MessageType("p2p-conn-data")
	MessageTypeP2PData              = network.MessageType("p2p-data")

	MessageTypeP2PRelayServers        = network.MessageType("p2p-relay-servers")
	MessageTypeP2PRelayValidate       = network.MessageType("p2p-relay-validate")
	MessageTypeP2PRelayAwait          = network.MessageType("p2p-relay-await")
	MessageTypeP2PRelayConnectionData = network.MessageType("p2p-relay-conn-data")
	MessageTypeP2PRelayPeersStatus    = network.MessageType("p2p-relay-peers-status")
	MessageTypeP2PRelayOpenStream     = network.MessageType("p2p-relay-open-stream")
	MessageTypeP2PRelayStreamInfo     = network.MessageType("p2p-relay-stream-info")
)

var (
	ErrorUnmarshalProto = errors.New("unmarshal proto model not found")
)

// Unmarshal data based on the MessageType
func Unmarshal(mtype network.MessageType) (body proto.Message, err error) {
	switch mtype {
	case MessageTypeClientValidate:
		body = &ClientValidateData{}
	case MessageTypeClientData:
		body = &ClientData{}
	case MessageTypeClientPing:
		body = &ClientPing{}
	case MessageTypeClientSearch:
		body = &ClientSearch{}
	case MessageTypeClients:
		body = &Clients{}

	case MessageTypeStreamConnectionData:
		body = &StreamConnectionData{}
	case MessageTypeStreamConnectionStatus:
		body = &StreamConnectionStatus{}

	case MessageTypeP2PClientInit:
		body = &NoDataMessage{}
	case MessageTypeP2PConnectionRequest:
		body = &P2PConnectionRequest{}
	case MessageTypeP2PConnectionStatus:
		body = &P2PConnectionStatus{}
	case MessageTypeP2PConnectionData:
		body = &P2PConnectionData{}
	case MessageTypeP2PData:
		body = &P2PData{}

	case MessageTypeP2PRelayServers:
		body = &P2PRelayServers{}
	case MessageTypeP2PRelayValidate:
		body = &ClientValidateData{}
	case MessageTypeP2PRelayAwait:
		body = &ClientValidateData{}
	case MessageTypeP2PRelayConnectionData:
		body = &P2PRelayConnectionData{}
	case MessageTypeP2PRelayPeersStatus:
		body = &P2PRelayPeersStatus{}
	case MessageTypeP2PRelayOpenStream:
		body = &P2PRelayOpenStream{}
	case MessageTypeP2PRelayStreamInfo:
		body = &P2PRelayStreamInfo{}

	default:
		err = fmt.Errorf("model: type(%v): %w", mtype, ErrorUnmarshalProto)
	}
	return
}
