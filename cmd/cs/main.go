package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/nyiyui/qanms/cs"
	"github.com/nyiyui/qanms/node"
	"github.com/nyiyui/qanms/node/api"
	"github.com/nyiyui/qanms/util"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type Config struct {
	CC     *node.CentralConfig `yaml:"central"`
	Tokens *Tokens             `yaml:"tokens"`
}

type Tokens struct {
	raw []cs.Token
}

func (t *Tokens) UnmarshalYAML(value *yaml.Node) error {
	var raw []Token
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	t2, err := convertTokens(raw)
	if err != nil {
		return err
	}
	t.raw = t2
	return nil
}

type Token struct {
	Name    string         `yaml:"name"`
	Hash    *util.HexBytes `yaml:"hash"`
	CanPull bool           `yaml:"can-pull"`
	CanPush bool           `yaml:"can-push"`
}

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
				Name:    token.Name,
				CanPull: token.CanPull,
				CanPush: token.CanPush,
			},
		}
	}
	return res, nil
}

var addr string
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
	for _, cn := range config.CC.Networks {
		if cn.Me != "" {
			return nil, fmt.Errorf("net %s: me is not blank")
		}
	}
	return &config, nil
}

func main() {
	flag.StringVar(&addr, "addr", "", "bind address")
	flag.StringVar(&configPath, "config", "", "config file path")
	flag.Parse()

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("load config: %s", err)
	}
	log.Print("config loaded")

	reloadCh := make(chan os.Signal, 1)
	signal.Notify(reloadCh, syscall.SIGHUP)

	server := cs.New(*config.CC)
	server.ReplaceTokens(config.Tokens.raw)
	go reloadOnCh(reloadCh, server)
	gs := grpc.NewServer()
	api.RegisterCentralSourceServer(gs, server)
	lis, err := net.Listen("tcp", addr)
	log.Print("聞きます…")
	if err != nil {
		log.Fatalf("listen: %s", err)
	}
	err = gs.Serve(lis)
	if err != nil {
		log.Fatalf("serve: %s", err)
	}
}

func reloadOnCh(reloadCh <-chan os.Signal, server *cs.CentralSource) {
	select {
	case <-reloadCh:
		cfg, err := loadConfig()
		if err != nil {
			log.Printf("sighup: load config: %s", err)
		}
		server.ReplaceTokens(cfg.Tokens.raw)
		server.ReplaceCC(cfg.CC)
		log.Printf("sighup: reloaded config")
	}
}
