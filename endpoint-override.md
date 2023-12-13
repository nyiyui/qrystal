# Endpoint Override

Nodes can individually override endpoints to proxy WireGuard connections.

This is achieved by speicyfing an executable path in `endpointOverride` setting in Node config.

The endpoint override executable (EOE) is executed an each time an endpoint is needed.
EOE reads a single line (use LF not CRLF) of JSON and returns a single line of JSON as a response, and must return 0 as the exit code.

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
  "endpoint": "127.0.0.1:51821" // this could be e.g. a proxy that bypasses a UDP firewall
}
```

Reification will block on EOE requests.
