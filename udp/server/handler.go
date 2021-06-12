package udps

import (
	"fmt"

	"github.com/supergiant-hq/xnet/model"
	"github.com/supergiant-hq/xnet/network"
)

func (c *Client) handleTick() {
	if c.Closed {
		c.ticker.Stop()
	}

	if !c.initialized {
		c.Close(401, "Client initialization incomplete")
	}
}

func (c *Client) handlePing(msg *network.Message) {
	c.log.Debugf("Ping from: %s", c.String())
}

func (c *Client) handleMessages(s *Server) {
	for {
		if c.Closed {
			return
		}
		msg, err := c.channel.Read(true)

		if err != nil {
			c.log.Warnln("Closing message stream:", c.String(), err.Error())
			c.Close(400, err.Error())
			return
		}

		switch msg.Ctx.Type {
		case model.MessageTypeClientValidate:
			if c.initialized {
				c.log.Warnln("Already initialized")
				continue
			}

			initData := msg.Body.(*model.ClientValidateData)
			clientData, err := s.validateClient(c, initData)
			if clientData == nil {
				clientData = &model.ClientData{}
			}
			if err == nil {
				clientData.Status = true
				clientData.Message = "Ok"
			} else {
				clientData.Status = false
				clientData.Message = err.Error()
			}
			c.sendInitStatus(msg, clientData)
			if err != nil {
				c.Close(401, fmt.Sprintf("Initialization Failed: %s", err.Error()))
				return
			}

			c.log.Infoln("Client initialized:", c.String())

			if s.clientConnectedHandler != nil {
				go s.clientConnectedHandler(c)
			}

			go c.handleStreams()

			c.log.Infoln("Client Connected:", c.String())

		case model.MessageTypeClientPing:
			c.handlePing(msg)

		case model.MessageTypeClientSearch:
			c.handleClientSearch(msg)

		default:
			if !c.initialized {
				c.Close(401, "Initialization incomplete")
				return
			}

			// Client Level Handler
			if handler, ok := c.messageHandler[msg.Ctx.Type]; ok {
				go handler(c, msg)
				continue
			}

			// Server Level Handler
			if handler, ok := s.messageHandler[msg.Ctx.Type]; ok {
				go handler(c, msg)
				continue
			}

			c.log.Warnln(fmt.Sprintf("Message handler not found: %v", msg.String()))
		}
	}
}

func (c *Client) sendInitStatus(msg *network.Message, clientData *model.ClientData) {
	rmsg, err := msg.GenReply(model.MessageTypeClientData, clientData)
	if err != nil {
		return
	}

	c.Send(rmsg)
}

func (c *Client) handleClientSearch(msg *network.Message) {
	var err error
	clients := []string{}

	c.log.Infoln("Search clients (%s)", c.Id)

	defer func() {
		rdata := &model.Clients{
			Clients: clients,
		}

		if err != nil {
			rdata.Status = false
			rdata.Message = err.Error()
			c.log.Errorf("Search clients error (%s): %v", c.Id, err.Error())
		} else {
			rdata.Status = true
			rdata.Message = "Ok"
			c.log.Infoln("Search clients res (%s): %d", c.Id, len(clients))
		}

		rmsg, _ := msg.GenReply(model.MessageTypeClients, rdata)
		c.Send(rmsg)
	}()

	params := msg.Body.(*model.ClientSearch)

	if len(params.Id) > 0 {
		client, cerr := c.server.GetClient(params.Id)
		if cerr != nil {
			err = cerr
			return
		}
		clients = append(clients, client.Id)
	} else if len(params.Tag) > 0 {
		sClients := c.server.GetClientsWithTag(params.Tag)
		if len(sClients) == 0 {
			err = fmt.Errorf("search yielded no results")
			return
		}
		for _, client := range sClients {
			clients = append(clients, client.Id)
		}
	} else {
		err = fmt.Errorf("invalid search params")
	}
}
