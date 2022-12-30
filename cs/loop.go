package cs

import (
	"crypto/tls"
	"fmt"

	"github.com/nyiyui/qrystal/util"
)

func (c *CentralSource) Handle(addr string, tlsCfg TLS) error {
	cert, err := tls.LoadX509KeyPair(tlsCfg.CertPath, tlsCfg.KeyPath)
	if err != nil {
		return fmt.Errorf("loading cert or key: %w", err)
	}
	util.S.Info("LoadX509KeyPair ok")
	cfg := tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := tls.Listen("tcp", addr, &cfg)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	util.S.Info("Listen ok")
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				util.S.Fatalf("Accept: %s", err)
			}
			go c.serve(conn)
		}
	}()
	util.S.Info("serving…")
	return nil
}
