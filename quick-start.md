# Quick Start

This assumes a systemd-based Linux installation.

## TLS

Use [minica](https://github.com/jsha/minica) to make TLS configuration:

```
$ minica -domains general.internal.example.net
$ minica -domains site.internal.example.net
```

## CS

A CS provides a Central configuration and coordinates forwarding between Nodes.

```yml
# general
addr: :39252
tls-cert-path: /etc/qrystal/cert.pem
tls-key-path: /etc/qrystal/key.pem
db-path: /etc/qrystal/db
tokens:
  - name: site
    hash: bf54fe3cbf6c49dd0ead501a47992d6cbb5632acb13879c738a16e726470c892
    can-pull: true
    networks:
      site-db: site
  - name: bastion
    hash: c094b71cbf95bbf39ef2008614b94e7eda19e528c5d47e647d63c4ceb71e988d
    can-pull: true
    networks:
      site-db: bastion
central:
  networks:
    site-db:
      ips:
        - 10.39.0.0/16
      listen-port: 39253
      keepalive: 30s
      peers:
        site:
          host: site.internal.example.net
          public-key: R_4hSCrgH1rSPKLYfchVDjhqFpDJW8bC8v5dXZH/hVH1A=
          allowed-ips:
            - 10.39.0.1/32
        bastion:
          host: # no host because bastion will not run a Node server.
          public-key: R_ylCnQi56ISXhtx7ftshi+Ri7znetZrQMGntzytpW0zs=
          allowed-ips:
            - 10.39.0.1/32
```

## Node

A node is configured using `/etc/qrystal/node-config.yml`. It can contain its own Central configuration, or it can dynamically import it from a CS.

```yml
# bastion
private-key: R_ylCnQi56ISXhtx7ftshi+Ri7znetZrQMGntzytpW0zs=
cs:
  host: general.internal.example.net:39252
  token: Fc/3byxjm2MICTepYSzOQ5Kao0OaY/xarOizqzPJU7ZKQPnnQ0kQeVU0Ez8WVPQgh8JnHfk+AIG5e6dUwiPI1A==
```

```yml
# site
server:
  tls-cert-path: /etc/qrystal/cert.pem
  tls-key-path: /etc/qrystal/key.pem
private-key: R_4hSCrgH1rSPKLYfchVDjhqFpDJW8bC8v5dXZH/hVH1A=
cs:
  host: general.internal.example.net:39252
  token: uA7y+O4OUNvHqBEUWf+yrvghVQtq2Jy6MhnIlY+9tWFNkjqqmVwB43jk9FwS6x/fn/yMTP1j+dQ6N0wA6aZJeA==
```

Additionally, a network needs at least one Node accessible from all others (double-forwarding is not supported).

A "client" Node will connect to any available "server" Nodes. A server Node requires a TLS keypair, such as with `site`. `bastion` will connect to `site`, but `site` will not.
