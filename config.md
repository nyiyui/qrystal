# Configuration

Each component (e.g. Node, CS, Azusa) can be configured independently so you
can gradually implement more as necessary.

## Node

```yaml
# Private key generated using gen-keys
private-key: R_…
server:
  # TLS cert and key path. Make sure the files are readable by qrystal-node user.
  tls-cert-path: /etc/qrystal/fullchain.pem
  tls-key-path: /etc/qrystal/privkey.pem
  # Address to bind to
  addr: :39251
# Central config (used if not provided by CS or before 1st CS pull); same as
# central in CS
central:
  …
# CS config (if central is not configured)
cs:
  host: qrystal.example.net:39252
  token: …
# Azusa config (if you want this Node to dynamically add peers at startup)
azusa:
  networks:
    sites: my-dev-machine
  # `host` field for peer. Leave blank for NAT-ed peers.
  host:
```

## CS

```yaml
addr: :39252
tokens:
  - name: my-dev-machine-azusa
    hash: …
    can-push: true
    # Networks and the peer names the token can push to, and can become peers for.
    networks:
      sites: my-dev-machine
      sysadmin: my-dev-machine
central:
  networks:
    sites:
      keepalive: 30s
      listen-port: 58120
      ips:
        - 10.123.0.1/16
      peers:
    sysadmin:
      keepalive: 30s
      listen-port: 58121
      ips:
        - 10.123.1.1/16
      peers:
        my-dev-machine:
          public-key: U_…
          # IPs always added to AllowedIPs. AllowedIPs can also include IP
          # ranges for forwarding.
          allowed-ips:
            - 10.123.1.2/32
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
