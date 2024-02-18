// Package testhost performs a test connection and handshake on the host, and a quick bandwidth and latency test.
package testhost

import (
	"net"
	"time"
)

type Result struct {
	// Connected is whether the test was able to connect and handshake the host.
	// If Connected is false, Bandwidth and Latency do not have meaning.
	Connected bool
	// Bandwidth in bits/second.
	Bandwidth uint
	Latency   time.Duration
}

func TestHost(endpoint *net.UDPAddr) Result {
	panic("not implemented yet")
}
