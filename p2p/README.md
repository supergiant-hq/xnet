# P2P (Peer-to-Peer) Network

This package provides a framework to create a P2P Network

## Modules

- _Broker_
  - _Server_
    - Manages connections between Broker Clients.
    - Right now, a network can only have one Broker Server.
    - This is by design as a Broker Server has the sole task of brokering between Clients. It does not act as a Relay.
  - _Client_
    - Standalone entity used to connect to other clients (peers).
- _Relay_
  - When a P2P connection cannot be established, the clients can request route traffic through a relay.
  - There can be multiple relays in a P2P network. A client choses the one closest to it (by pinging) to relay it's connection.
  - A client can also use a predefined relay instead of dynamically choosing one closest to it.
- _Server_
  - Used in _Broker - Server_
  - Helps in establishing P2P connections between clients.
- _Client_
  - Used in _Broker - Client_
  - Manages P2P connections with other clients.

## Examples

- [xTUNNEL][clixtunnel] - Tunnel TCP/UDP traffic between nodes

[//]: # "Links"
[clixtunnel]: https://github.com/supergiant-hq/xtunnel
