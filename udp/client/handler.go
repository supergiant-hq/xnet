package udpc

import (
	"fmt"
	"time"

	"github.com/supergiant-hq/xnet/network"

	"github.com/supergiant-hq/xnet/model"
)

func (c *Client) handleInitData(data *model.ClientData) (err error) {
	c.Data = data

	if !data.Status {
		c.log.Errorf("Initialization failed: %s", data.Message)
		err = fmt.Errorf(data.Message)
		return
	}

	c.Id = data.Id
	c.init = true

	c.log.Infof("Initialization complete: %s", c.String())
	return
}

func (c *Client) handlePings(sessionId string) {
	defer func() {
		recover()
	}()

	for {
		msg := network.NewMessage(
			model.MessageTypeClientPing,
			&model.ClientPing{
				Time: time.Now().String(),
			},
		)
		_, err := c.Send(msg)
		if err != nil {
			c.log.Warnf("Error sending ping to server: %s", err.Error())
			return
		}

		if c.Closed || c.sessionId != sessionId {
			return
		}
		<-time.After(time.Second * 15)
	}
}

func (c *Client) handleMessages(sessionId string) {
	defer func() {
		recover()
		if !c.Closed {
			c.reconnect()
		}
	}()

	for {
		if c.Closed {
			return
		}

		msg, err := c.channel.Read(true)
		if err != nil {
			c.log.Warnln("Closing message stream:", c.String(), err.Error())
			return
		}

		handler, ok := c.messageHandler[msg.Ctx.Type]
		if !ok {
			c.log.Warnf("Message handler not found for (%v)", msg.Ctx.Type)
			continue
		}
		go handler(c, msg)
	}
}

// Search for Clients based on certain parameters
func (c *Client) SearchClients(params *model.ClientSearch) (clients []string, err error) {
	msg := network.NewMessageWithAck(
		model.MessageTypeClientSearch,
		params,
		network.RequestTimeout,
	)
	rmsg, err := c.Send(msg)
	if err != nil {
		return
	}

	rdata := rmsg.Body.(*model.Clients)
	if !rdata.Status {
		err = fmt.Errorf(rdata.Message)
		return
	}
	clients = rdata.Clients

	return
}
