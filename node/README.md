Node does most of the work managing networks. It:
- Generates WireGuard configuration (applied using Mio)
- Fetches config from Centralsource
- Decides whether forwarding is necessary
- Contact other nodes to exchange WireGuard keys
- etc.
