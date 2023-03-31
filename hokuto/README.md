# Hokuto

Per-Node DNS server.

Domain format: `<peer-name>.<cn-name>.<parent>` e.g. `desktop.mynet.qrystal.internal`

Returns `A` records per-peers' allowed IPs, and `TXT` records for other metadata.

This DNS server **doesn't** forward unknown DNS requests to some server.

## TODO

- Reverse lookups.
