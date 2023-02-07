//go:build !linux

package mio

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func Listen() (lis net.Listener, addr string, err error) {
	lis, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return
	}
	addr = fmt.Sprintf("tcp:127.0.0.1:%d", strconv.Itoa(lis.Addr().(*net.TCPAddr).Port))
	return
}

// Guard aborts connections from hosts that are not localhost.
//
// This is intended to be a last resort; it should not be relied on for
// security.
//
// Note: Server should be listening on 127.0.0.1, not 0.0.0.0.
func Guard(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.RemoteAddr, "127.0.0.1:") {
			log.Printf("blocked request from %s", r.RemoteAddr)
			http.Error(w, "", 403)
			return
		}
		handler.ServeHTTP(w, r)
	})
}
