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
	KEY_PORT          = "PORT"
	KEY_CONNECTION_ID = "CONNECTION_ID"

	KEY_STREAM_IGNORE  = "STREAM_IGNORE"
	KEY_STREAM_MESSAGE = "STREAM_MESSAGE"
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
