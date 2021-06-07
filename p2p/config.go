package p2p

import (
	"time"
)

const (
	TickerDuration    = time.Second * 15
	RequestTimeout    = time.Second * 5
	ConnectionTimeout = time.Second * 15
)

const (
	TAG_RELAY = "relay"
)

const (
	KEY_CONNECTION_ID   = "__CONNECTION_ID"
	KEY_PORT            = "__PORT"
	KEY_INITIATOR       = "__INITIATOR"
	KEY_STREAM_INTERNAL = "__STREAM_INTERNAL"
)

type ConnectionMode string

const (
	ConnectionModeRelay ConnectionMode = "relay"
	ConnectionModeP2P   ConnectionMode = "p2p"
)

type ConnectionState int

const (
	ConnectionStateDisconnected ConnectionState = iota
	ConnectionStateConnecting
	ConnectionStateConnected
)
