# xNET

[![ProjectStatus](https://img.shields.io/badge/status-experimental-orange)](README.md)

xNET is a framework to build tools as simple as a network tunnel or as complex as an overlay network

```sh
go get github.com/supergiant-hq/xnet
```

## Modules

- [Generic UDP Client and Server][udpreadme] using QUIC protocol
- [P2P Network][p2preadme] with Broker, Relay and Client implementations
- TUN Device for Linux, Darwin and Windows (TODO)

## Packages

This framework depends on the following core dependencies

| Module                 | Link            |
| ---------------------- | --------------- |
| lucas-clemente/quic-go | [View][pkgquic] |
| songgao/water          | [View][pkgtun]  |
| go-ping/ping           | [View][pkgping] |

## Examples

- [xTunnel][clixtunnel] - Tunnel TCP/UDP traffic between nodes

## License

Apache License 2.0

[//]: # "Links"
[udpreadme]: https://github.com/supergiant-hq/xnet/tree/master/udp
[p2preadme]: https://github.com/supergiant-hq/xnet/tree/master/p2p
[pkgquic]: https://github.com/lucas-clemente/quic-go
[pkgtun]: https://github.com/songgao/water
[pkgping]: https://github.com/go-ping/ping
[clixtunnel]: https://github.com/supergiant-hq/xtunnel
