package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/nyiyui/qrystal/cs"
	"github.com/nyiyui/qrystal/node/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v3"
)

func convertTokens(tokens []Token) ([]cs.Token, error) {
	res := make([]cs.Token, len(tokens))
	for i, token := range tokens {
		var hash [sha256.Size]byte
		log.Println(len(hash))
		n := copy(hash[:], *token.Hash)
		if n != len(hash) {
			return nil, fmt.Errorf("token %d: invalid length (%d) hash", i, n)
		}
		res[i] = cs.Token{
			Hash: hash,
			Info: cs.TokenInfo{
				Name:         token.Name,
				Networks:     token.Networks,
				CanPull:      token.CanPull,
				CanPush:      token.CanPush,
				CanAddTokens: token.CanAddTokens,
			},
		}
	}
	return res, nil
}

var configPath string

func loadConfig() (*Config, error) {
	raw, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config read: %s", err)
	}
	var config Config
	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		return nil, fmt.Errorf("config unmarshal: %s", err)
	}
	for cnn, cn := range config.CC.Networks {
		if cn.Me != "" {
			return nil, fmt.Errorf("net %s: me is not blank", cnn)
		}
	}
	return &config, nil
}

func main() {
	flag.StringVar(&configPath, "config", "", "config file path")
	flag.Parse()

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("load config: %s", err)
	}
	log.Print("config file loaded")
	creds, err := credentials.NewServerTLSFromFile(config.TLSCertPath, config.TLSKeyPath)
	if err != nil {
		log.Fatalf("server tls: %s", err)
	}

	server := cs.New(*config.CC)
	server.ReplaceTokens(config.Tokens.raw)
	gs := grpc.NewServer(grpc.Creds(creds))
	api.RegisterCentralSourceServer(gs, server)
	lis, err := net.Listen("tcp", config.Addr)
	log.Print("聞きます…")
	if err != nil {
		log.Fatalf("listen: %s", err)
	}
	err = gs.Serve(lis)
	if err != nil {
		log.Fatalf("serve: %s", err)
	}
}
