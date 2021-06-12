# xNET

xNET is a generic network framework to build tools as simple as a network tunnel or as complex as an overlay network.

## Modules

- [Generic UDP Client and Server][udpreadme] using QUIC protocol
- [P2P Network][p2preadme] with Broker, Relay and Client implementations
- TUN Device for Linux, Darwin and Windows (TODO)

## Packages

xNET depends on the following core packages

| Module                 | Link            |
| ---------------------- | --------------- |
| lucas-clemente/quic-go | [View][pkgquic] |
| songgao/water          | [View][pkgtun]  |
| go-ping/ping           | [View][pkgping] |

## Examples

- [xTUNNEL][clixtunnel] - Tunnel TCP/UDP traffic between nodes

## TODO

- Finish TUN module
- P2P auto failover to relay mode

## License

Apache License 2.0

[//]: # "Links"
[udpreadme]: https://github.com/supergiant-hq/xnet/tree/master/udp
[p2preadme]: https://github.com/supergiant-hq/xnet/tree/master/p2p
[pkgquic]: https://github.com/lucas-clemente/quic-go
[pkgtun]: https://github.com/songgao/water
[pkgping]: https://github.com/go-ping/ping
[clixtunnel]: https://github.com/supergiant-hq/xtunnel
