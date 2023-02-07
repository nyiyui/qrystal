# Hokuto

DNS server for a Node.

Domain format: `<peer-name>.<cn-name>.<parent>` e.g. `matsuri.kaii.gensokyo.mcpt.ca`

Returns `A` records per-peers' allowed IPs, and `TXT` records for other metadata.

## TODO

- Initial protocol to fetch networks from qrystal-node.
- Restrict returned domains to only which the client can see.
- Forward DNS requests.
- Reverse lookups.
