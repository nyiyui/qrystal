package main

import (
	"log"
	"net"

	"github.com/nyiyui/qanms/cs"
	"github.com/nyiyui/qanms/node/api"
	"google.golang.org/grpc"
)

type Config struct {
	Addr string `yaml:"addr"`
}

func main() {
	var c Config

	server := cs.New()
	gs := grpc.NewServer()
	api.RegisterCentralSourceServer(gs, server)
	lis, err := net.Listen("tcp", c.Addr)
	if err != nil {
		log.Fatalf("listen: %s", err)
	}
	err = gs.Serve(lis)
	if err != nil {
		log.Fatalf("serve: %s", err)
	}
}
