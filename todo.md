# TODO

## Host Fallback

Specify multiple hosts for peers.

Example
```yaml
central:
  networks:
    examplenet:
      …
      peers:
        vps0:
          public-key: …
            hosts:
              - qrystal.vps0.example.net:39251
              - 123.45.67.89:39251
```
