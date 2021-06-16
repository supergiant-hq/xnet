package network

import "errors"

var (
	ErrorPanic         = errors.New("panic")
	ErrorTimeout       = errors.New("timeout")
	ErrorChannelClosed = errors.New("channel closed")
)

var (
	ErrorGenRes = errors.New("message does not need a response")
)
