package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/node"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/profile"
	"github.com/nyiyui/qrystal/util"
	"gopkg.in/yaml.v3"
)

type config struct {
	PrivKey  string         `yaml:"private-key"`
	Server   *serverConfig  `yaml:"server"`
	Central  central.Config `yaml:"central"`
	CS       *csConfig      `yaml:"cs"`
	CS2      []csConfig     `yaml:"cs2"`
	Azusa    *azusaConfig   `yaml:"azusa"`
	Kiriyama string         `yaml:"kiriyama"`
}

type serverConfig struct {
	TLSCertPath string `yaml:"tls-cert-path"`
	TLSKeyPath  string `yaml:"tls-key-path"`
	Addr        string `yaml:"addr"`
}

type azusaConfig struct {
	Networks map[string]string `yaml:"networks"`
	Host     string            `yaml:"host"`
}

type configValidated config

func (c *configValidated) UnmarshalYAML(value *yaml.Node) error {
	var c2 config
	err := value.Decode(&c2)
	if err != nil {
		return err
	}
	if len(c.PrivKey) == 0 {
		return errors.New("private-key cannot be blank")
	}
	if c.PrivKey[0] != 'R' {
		return errors.New("private-key is not a private key (starts with \"R\")")
	}
	*c = configValidated(c2)
	return nil
}

type csConfig struct {
	Comment     string       `yaml:"comment"`
	TLSCertPath string       `yaml:"tls-cert-path"`
	AllowedNets []string     `yaml:"networks"`
	Host        string       `yaml:"host"`
	Token       string       `yaml:"token"`
	Azusa       *azusaConfig `yaml:"azusa"`
}

func processCSConfig(cfg *csConfig) (*node.CSConfig, error) {
	var creds credentials.TransportCredentials
	var err error
	if cfg.TLSCertPath == "" {
		creds = credentials.NewTLS(nil)
	} else {
		creds, err = credentials.NewClientTLSFromFile(cfg.TLSCertPath, "")
		if err != nil {
			return nil, fmt.Errorf("tls cert: %w", err)
		}
	}
	netsAllowed := make([]*regexp.Regexp, len(cfg.AllowedNets))
	for i, net := range cfg.AllowedNets {
		netsAllowed[i], err = regexp.Compile(net)
		if err != nil {
			return nil, fmt.Errorf("network %d: %w", i, err)
		}
	}
	return &node.CSConfig{
		Comment:         cfg.Comment,
		Creds:           creds,
		Host:            cfg.Host,
		Token:           cfg.Token,
		NetworksAllowed: netsAllowed,
		//Azusa:cfg.Azusa, TODO: azusa
	}, err
}

func main() {
	util.SetupLog()

	log.SetPrefix("node: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	util.ShowCurrent()
	profile.Profile()

	var c config
	data, err := ioutil.ReadFile(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatalf("read config: %s", err)
	}
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		log.Fatalf("load config: %s", err)
	}

	privKey, err := base64.StdEncoding.DecodeString(c.PrivKey[2:])
	if err != nil {
		log.Fatalf("load config: decoding private key failed: %s", err)
	}
	privKey2 := ed25519.NewKeyFromSeed([]byte(privKey))

	var serverCreds credentials.TransportCredentials
	if c.Server != nil {
		s := c.Server
		serverCreds, err = credentials.NewServerTLSFromFile(s.TLSCertPath, s.TLSKeyPath)
		if err != nil {
			log.Fatalf("client tls: %s", err)
		}
	}

	// CS
	ncscs := make([]node.CSConfig, 1+len(c.CS2))
	ncsc, err := processCSConfig(c.CS)
	if err != nil {
		log.Fatalf("config cs: %s", err)
	}
	ncscs[0] = *ncsc
	for i, csc := range c.CS2 {
		ncsc, err := processCSConfig(&csc)
		if err != nil {
			log.Fatalf("config cs2 %d: %s", i, err)
		}
		ncscs[1+i] = *ncsc
		if csc.Azusa != nil {
			ncsc.Azusa = &node.AzusaConfig{
				Host:     csc.Azusa.Host,
				Networks: csc.Azusa.Networks,
			}
		}
	}

	mioAddr := os.Getenv("MIO_ADDR")
	mioToken, err := base64.StdEncoding.DecodeString(os.Getenv("MIO_TOKEN"))
	if err != nil {
		log.Fatalf("parse MIO_TOKEN: %s", err)
	}
	n, err := node.NewNode(node.NodeConfig{
		PrivKey:  privKey2,
		CC:       c.Central,
		MioAddr:  mioAddr,
		MioToken: mioToken,
		CS:       ncscs,
	})
	if err != nil {
		panic(err)
	}

	if c.CS == nil && c.Azusa != nil {
		log.Fatal("config: azusa requires cs")
	}

	if c.Server != nil {
		go func() {
			server := grpc.NewServer(grpc.Creds(serverCreds))
			api.RegisterNodeServer(server, n)
			lis, err := net.Listen("tcp", c.Server.Addr)
			if err != nil {
				log.Fatal(err)
			}
			err = server.Serve(lis)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}
	if c.CS != nil {
		if c.Azusa != nil {
			n.AzusaConfigure(c.Azusa.Networks, c.Azusa.Host)
		}
		go n.ListenCS()
	}
	if c.Kiriyama != "" {
		n.Kiriyama.Addr = c.Kiriyama
		go n.Kiriyama.Loop()
	}
	select {}
}
