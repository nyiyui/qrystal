# Configuration

Each component (e.g. Node, CS, Azusa) can be configured independently so you
can gradually implement more as necessary.

## Central

```yaml
central:
  networks:
    # Each network's peers should be fully connected.
    toaru: # must be an allowed WireGuard interface name (i.e. `[a-zA-Z0-9_=+.-]{1,15}`)
      # Settings for all peers
      keepalive: 30s # = Keepalive (see https://pkg.go.dev/time#ParseDuration )
      listen-port: 58120 # = ListenPort
      # Allowed IPs for the entire network.
      ips:
        - 10.123.0.1/16
      # Peer name for this Node (if applicable). 
      me: mikoto
      peers:
        mikoto:
          # Address of Node available from all peers' WGs, if this Node runs a server.
          host: mikoto.example.net:39251
          
          # IPs always added to AllowedIPs. Peers that this Node forwards should be added to forwarding-peers. Forwarding without CS may not work.
          allowed-ips:
            - 10.123.1.2/32
          forwarding-peers: []
          
          # Public key of the Node (corresponds to private-key).
          public-key: U_…
          
          # NOTE: can-forward is unused (i.e. is junk).
```

## Node

```yaml
# Private key generated using gen-keys.
private-key: R_…

# If this Node runs a server:
server:
  # TLS cert and key path. Make sure the files are readable by `qrystal-node` user.
  tls-cert-path: /etc/qrystal/fullchain.pem
  tls-key-path: /etc/qrystal/privkey.pem
  
  # Address to bind to
  addr: :39251

# Central config (used if not provided by CS or before 1st CS pull).
central:

# CS config
cs:
  # TLS cert to use to connect.
  tls-cert-path:
  
  # Networks to apply from this CS.
  networks:
  
  # CS address.
  host: qrystal.example.net:39252
  
  # Token to pull/push.
  token: …
  
  # Azusa config.
  azusa:

cs2: [] # CS config but multiple.

# Azusa config (if you want this Node to dynamically add peers at startup to cs, not cs2).
azusa:
  networks:
    toaru: kuroko
  # `host` field for peer. Leave blank for NAT-ed peers.
  host: kuroko.example.net:39251
```

## CS

```yaml
addr: :39252
# TLS settings.
tls-cert-path:
tls-key-path:

# Path to backport to.
backport-path: /etc/qrystal/cs-backport.yml

# Database for (currently) tokens.
db-path: /etc/qrystal/db

# Tokens to add/overwrite to db on startup.
tokens:
  - # Name for humans.
    name: my-dev-machine-azusa
  
    # Token hash.
    hash: …
    
    can-push:
      # Can push to any  peer on any network.
      any: true
      # Networks and the peer names the token can be used to push to. Null to disallow pushing.
      networks:
        <cnn>: <pn>
    
    can-pull: true
    # Networks and the peer names the token can pull for. Useless if `can-pull: false`.
    networks:
      sites: my-dev-machine
      sysadmin: my-dev-machine
    
    can-add-tokens:
      # Whether the token can add tokens that can push (unrestricted).
      can-push: true | false
      # Whether the token can add tokens that can pull (unrestricted).
      can-pull: true | false

# Central configuration.
central:
```

## Example

- centralsource runs on `main.example.net:39252`
- node runs on `{database, main}.example.net:39251`

```yaml
# cs-config.yml on main
addr: :39252
tokens:
  - name: main-token
    hash: 2907d8a8fa43a530d6477aacb7f8577fbbde6c49d9122880b46394fba5fa273a
    can-pull:
    networks:
      examplenet: main
  - name: database-token
    hash: f092b3c58c16d22a8d4d30a0d776fa0a877a434079b91c9bd8eac900b481f4ba
    can-pull: true
    networks:
      examplenet: database
  - name: sysadmin-nyiyui
    hash: 34afd1a01f6e679403ee42badd460b5bfa95487a7660a23c960d361b7626a02b
    can-add-tokens:
      can-pull: true # required to add tokens with can-pull: true
central:
  networks:
    examplenet:
      keepalive: 10s
      listen-port: 58120
      ips:
        - 10.123.0.1/16
      peers:
        main:
          public-key: U_vy3etraFR+jTMgRZ95PIBTc43HUsRmMREijN1cP/tH4=
          allowed-ips:
            - 10.123.0.1/32
        database:
          public-key: U_/0nqDH7s6e+O5dgixFBRIHMM+GsFXVDxgaCG+6LpTi0=
          allowed-ips:
            - 10.123.0.2/32
```

```yaml
# node-config.yml on main
private-key: R_iTwh5DNRokc8J2iSZHkPgSJkSAM7CdX2QP0PFbrZBoU=
addr: :39251
cs:
  host: main.example.net:39252
  token: 54haXJnLrts59/PNZBQUobzu71fEaiinMrTaBnOtmg208+uDA1cvndkiVKVABmdLxmOF8YjCAfZoiM74ioMEeg==
```

```yaml
# node-config.yml on database
private-key: R_C0MatgWVGXquCEGjQH60jnL9imUAK6N3knVfSpjt9q4=
addr: :39251
cs:
  host: main.example.net:39252
  token: /ghrTpjdIRqQJQdUQiNfabmJncTl5KN7nukLTyXZwSLp1rWi/C9OTXVrs8WMYAQ/aNqvc1lr4Xcr2gj9PlyUow==
```

```yaml
# cs-push-config.yml on sysadmin's computer
server: main.example.net:39252
token: 8qYQRldGIwXB98lCqe/VqeOp1NJ/lN+tM+mUDfdqjdZabsIWYiD0ru6nINe02C5XHlrkXJByLZXM7Q1SFvyKnQ==
```

```yaml
# cs-push-tmp-config.yml on sysadmin's computer (generated per invocation)
config-path: /home/nyiyui/cs-push-config.yml
overwrite: false
name: backend-0
networks:
  examplenet: backend-0
public-key: U_pm21oL5DQBOGNYB6vkhNGr9uTHMP1t12+9YbOt9a0jg=
# backend-0 would have R_MIcsEQM2LoZPBuoCUioloWUjd5YINY/gc5uzyOSJO3M=
token-hash: cdf3e19494bdae98cfa6e0b72fde8714eb4bfff786209311d5594d4c994d7c71
# backend-0 would have Vlm2XR03W8P1N2ksw3NCPz6pjifSkVEXRpHZIWYT3IegGrv+kEpeK0WwUWiwxfEO1zGeFhbI6XZbujFn8xjonQ==
```
