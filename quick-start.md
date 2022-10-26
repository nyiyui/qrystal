# Quick Start

This assumes a systemd-based Linux installation.

## CS

A CS provides a Central configuration and coordinates forwarding between Nodes.

```yml
addr: :39252
tls-cert-path:
tls-key-path:
central:
db-path: /etc/qrystal/db
```

## Node

A node is configured using `/etc/qrystal/node-config.yml`. It can contain its own Central configuration, or it can dynamically import it from a CS.

```yml
private-key:
central:
cs:
  host:
  token:
```

Additionally, a network needs at least one Node accessible from all others (double-forwarding is not supported).

A "client" Node will connect to any available "server" Nodes. A server Node requires a TLS keypair:

```yml
server:
  tls-cert-path:
  tls-key-path:
```

## Central Configuration

TODO
