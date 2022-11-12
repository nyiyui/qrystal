package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/nyiyui/qrystal/cs"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/profile"
	"github.com/nyiyui/qrystal/util"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var configPath string

func main() {
	flag.StringVar(&configPath, "config", "", "config file path")
	flag.Parse()

	util.L, _ = zap.NewDevelopment()
	defer util.L.Sync()
	util.S = util.L.Sugar()

	util.ShowCurrent()
	profile.Profile()

	config, err := cs.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("load config: %s", err)
	}
	log.Print("config file loaded")
	creds, err := credentials.NewServerTLSFromFile(config.TLSCertPath, config.TLSKeyPath)
	if err != nil {
		log.Fatalf("server tls: %s", err)
	}
	log.Printf("TLS creds read")

	db, err := buntdb.Open(config.DBPath)
	if err != nil {
		log.Fatalf("open db: %s", err)
	}
	server, err := cs.New(*config.CC, config.BackportPath, db)
	if err != nil {
		log.Fatalf("new: %s", err)
	}
	err = warnDivergentTokens(config, server)
	if err != nil {
		log.Fatalf("warn divergent tokens: %s", err)
	}
	err = server.AddTokens(config.Tokens.Raw)
	if err != nil {
		log.Fatalf("add tokens: %s", err)
	}
	if config.BackportPath != "" && false {
		err = server.ReadBackport()
		if err != nil {
			log.Fatalf("read backport: %s", err)
		}
		log.Printf("read backport from %s", config.BackportPath)
	}
	gs := grpc.NewServer(grpc.Creds(creds))
	api.RegisterCentralSourceServer(gs, server)
	lis, err := net.Listen("tcp", config.Addr)
	if err != nil {
		log.Fatalf("listen: %s", err)
	}
	log.Print("will serve…")
	err = gs.Serve(lis)
	if err != nil {
		log.Fatalf("serve: %s", err)
	}
}

// warnDivergentTokens warns for any divergent tokens.
func warnDivergentTokens(config *cs.Config, server *cs.CentralSource) error {
	for _, tr := range config.Tokens.Raw {
		already, ok, err := server.Tokens.GetTokenByHash(hex.EncodeToString(tr.Hash[:]))
		if err != nil {
			return fmt.Errorf("get token %x: %s", tr.Hash[:], err)
		}
		if !ok {
			continue
		}
		info2, err := json.Marshal(tr.Info)
		if err != nil {
			return fmt.Errorf("marshal token2 %x: %s", tr.Hash[:], err)
		}
		already2, err := json.Marshal(already)
		if err != nil {
			return fmt.Errorf("marshal token2 %x: %s", tr.Hash[:], err)
		}
		if !bytes.Equal(info2, already2) {
			util.S.Warnf("token %x diverges from db", tr.Hash[:])
		}
	}
	return nil
}
