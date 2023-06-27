# Endpoint Override

Nodes can individually override endpoints for proxying WireGuard UDP packets.

This is achieved by speicyfing an executable path in `endpointOverride` setting in Node config.

The endpoint override executable (EOE) is executed an unspecified amount of times (usually once) during the lifetime of the Node.
EOE reads a single line (please use LF not CRLF) of JSON and returns a single line of JSON as a response.

The request JSON (sent to EOE) is of the following format:
```javascript
{
  "cnn": "wg0",
  "pn": "server0",
  "endpoint": "server.example.com:51820"
}
```

The response JSON (sent from EOE) is of the following format:
```javascript
{
  "endpoint": "127.0.0.1:51821" // e.g. this could be a proxy that bypasses a UDP firewall
}
```

Reification will block on EOE requests.
