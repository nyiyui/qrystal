package cs

import (
	"crypto/tls"
	"fmt"

	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"google.golang.org/grpc"
)

func (c *CentralSource) HandleHaruka(addr string, tlsCfg TLS) error {
	util.S.Info("haruka: starting…")
	cert, err := tls.LoadX509KeyPair(tlsCfg.CertPath, tlsCfg.KeyPath)
	if err != nil {
		return fmt.Errorf("loading cert or key: %w", err)
	}
	util.S.Info("haruka: LoadX509KeyPair ok")
	cfg := tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := tls.Listen("tcp", addr, &cfg)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s := grpc.NewServer()
	api.RegisterCentralSourceServer(s, c)
	util.S.Infof("haruka: listen on %s", listener.Addr())
	go func() {
		err := s.Serve(listener)
		if err != nil {
			util.S.Fatalf("haruka: serve failed: %s", err)
		}
	}()
	util.S.Info("haruka: serving")
	return nil
}
