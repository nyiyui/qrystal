# Hokuto

Per-Node DNS server.

Domain format: `<peer-name>.<cn-name>.<parent>` e.g. `desktop.mynet.qrystal.internal`

Returns `A` records per-peers' allowed IPs, and `TXT` records for other metadata.

This DNS server **doesn't** forward unknown DNS requests to some server.
(This is to keep the program simple.)
Therefore, you need another DNS server in front of this one that only forwards requests the end in a specific parent (e.g. `.qrystal.internal`).

## TODO

- Reverse lookups.
- multiple parents (e.g. `.internal.example.org` and `.qrystal.internal)
- preset CNs for parents (e.g. `<peer-name>.internal.example.org` (no CN name))
