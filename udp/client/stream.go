package udpc

import (
	"context"
	"fmt"

	"github.com/supergiant-hq/xnet/network"
	"github.com/supergiant-hq/xnet/udp"
)

func (c *Client) handleStreams(sessionId string) {
	defer func() {
		recover()
	}()

	for {
		if c.Closed || c.sessionId != sessionId {
			return
		}

		stream, err := c.session.AcceptStream(context.Background())
		if err != nil {
			c.log.Debugln("Error accepting stream:", err.Error())
			continue
		}
		c.log.Infoln("Incoming stream...")
		if c.streamHandler == nil {
			c.log.Errorln("Error accepting stream:", "Client does not have a streamHandler")
			continue
		}

		cstream, err := udp.HandleStream(c.log.Logger, stream, c.Cfg.Unmarshalers())
		if err != nil {
			c.log.Errorln("Error accepting stream:", err.Error())
			continue
		}
		c.streams.Store(cstream.Id, cstream)
		c.log.Infoln("Accepted stream:", cstream.String())

		go c.streamHandler(c, cstream)
	}
}

// Open a new Stream to the Server
func (c *Client) OpenStream(data map[string]string) (cstream *udp.Stream, err error) {
	c.log.Info("Opening stream to: ", c.Cfg.ServerAddr.String())
	if !c.Connected {
		err = fmt.Errorf("not connected")
		return
	}

	stream, err := c.session.OpenStream()
	if err != nil {
		return
	}

	channel := network.NewChannel(c.log.Logger, stream, c.Cfg.Unmarshalers())
	cstream, err = udp.NewStream(data, channel)
	if err != nil {
		return
	}
	c.streams.Store(cstream.Id, cstream)
	c.log.Info("Opened stream: ", cstream.String())

	return
}

// Return the Stream associated with the ID
func (c *Client) GetStream(id string) (stream *udp.Stream, err error) {
	rstream, ok := c.streams.Load(id)
	if !ok {
		err = fmt.Errorf("stream not found")
		return
	}
	stream = rstream.(*udp.Stream)
	return
}

// Close the Stream associated with the ID
func (c *Client) CloseStream(id string) {
	if rstream, ok := c.streams.LoadAndDelete(id); ok {
		rstream.(*udp.Stream).Close()
	}
}

// Close all Streams
// Used when closing the Client
func (c *Client) CloseAllStreams() {
	c.streams.Range(func(key, value interface{}) bool {
		if stream, ok := c.streams.LoadAndDelete(key); ok {
			stream.(*udp.Stream).Close()
		}
		return true
	})
}
