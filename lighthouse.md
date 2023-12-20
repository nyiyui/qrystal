# Lighthouse - network discovery

A node might be on the same LAN (or otherwise be directly accesible, e.g. via the Internet).
In this case, we want to automatically discover which IP addresses belong to which node.

For a LAN:
- find devices via mDNS
- for each, use a separate WireGuard configuration to connect with the candiate endpoint
- send a few HTTP requests back-and-forth to test connectivity
