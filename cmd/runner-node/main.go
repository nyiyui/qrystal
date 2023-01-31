package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"

	"crypto/tls"
	"crypto/x509"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/node"
	"github.com/nyiyui/qrystal/profile"
	"github.com/nyiyui/qrystal/util"
	"gopkg.in/yaml.v3"
)

type config struct {
	CSs      []csConfig `yaml:"css"`
	Kiriyama string     `yaml:"kiriyama"`
}

type csConfig struct {
	Comment string `yaml:"comment"`
	TLS     struct {
		CertPath string `yaml:"certPath"`
	} `yaml:"tls"`
	AllowedNets []string   `yaml:"networks"`
	Host        string     `yaml:"endpoint"`
	Token       util.Token `yaml:"token"`
	TokenPath   string     `yaml:"tokenPath"`
	Azusa       struct {
		Networks map[string]central.Peer
	} `yaml:"azusa"`
}

func processCSConfig(cfg *csConfig) (*node.CSConfig, error) {
	var err error
	var cert []byte
	var tlsCfg *tls.Config
	if cfg.TLS.CertPath != "" {
		cert, err = os.ReadFile(cfg.TLS.CertPath)
		if err != nil {
			return nil, fmt.Errorf("read tls cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(cert) {
			return nil, fmt.Errorf("load pem: %w", err)
		}
		tlsCfg = &tls.Config{RootCAs: pool}
	}
	netsAllowed := make([]*regexp.Regexp, len(cfg.AllowedNets))
	for i, net := range cfg.AllowedNets {
		netsAllowed[i], err = regexp.Compile(net)
		if err != nil {
			return nil, fmt.Errorf("network %d: %w", i, err)
		}
	}
	if cfg.Token.Empty() {
		token, err := os.ReadFile(cfg.TokenPath)
		if err != nil {
			return nil, fmt.Errorf("load tokenPath %s: %s", cfg.TokenPath, err)
		}
		tok, err := util.ParseToken(string(token))
		if err != nil {
			return nil, fmt.Errorf("parse token: %s", err)
		}
		cfg.Token = *tok
	}
	return &node.CSConfig{
		Comment:         cfg.Comment,
		TLSConfig:       tlsCfg,
		Host:            cfg.Host,
		Token:           cfg.Token,
		NetworksAllowed: netsAllowed,
		Azusa:           cfg.Azusa.Networks,
	}, err
}

func main() {
	util.SetupLog()

	log.SetPrefix("node: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	util.ShowCurrent()
	profile.Profile()

	var c config
	configPath := os.Getenv("CONFIG_PATH")
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("read config: %s", err)
	}
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		log.Fatalf("load config: %s", err)
	}
	log.Printf("config loaded from %s: %#v", configPath, c)

	// CS
	ncscs := make([]node.CSConfig, 0, len(c.CSs))
	for i, csc := range c.CSs {
		ncsc, err := processCSConfig(&csc)
		if err != nil {
			log.Fatalf("config cs2 %d: %s", i, err)
		}
		ncscs = append(ncscs, *ncsc)
	}

	mioAddr := os.Getenv("MIO_ADDR")
	mioToken, err := base64.StdEncoding.DecodeString(os.Getenv("MIO_TOKEN"))
	if err != nil {
		log.Fatalf("parse MIO_TOKEN: %s", err)
	}
	n, err := node.NewNode(node.NodeConfig{
		MioAddr:  mioAddr,
		MioToken: mioToken,
		CS:       ncscs,
	})
	if err != nil {
		panic(err)
	}

	if c.CSs != nil {
		go n.ListenCS()
	}
	if c.Kiriyama != "" {
		n.Kiriyama.Addr = c.Kiriyama
		go n.Kiriyama.Loop()
	}
	select {}
}
