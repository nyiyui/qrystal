package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/nyiyui/qanms/cs"
	"github.com/nyiyui/qanms/node/api"
	"github.com/nyiyui/qanms/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       string `yaml:"server"`
	CentralToken string `yaml:"token"`
	TLSCertPath  string `yaml:"tls-cert-path"`
}

type TmpConfig struct {
	ConfigPath string                   `yaml:"config-path"`
	Overwrite  bool                     `yaml:"overwrite"`
	Name       string                   `yaml:"name"`
	Networks   map[string]NetworkConfig `yaml:"networks"`

	PublicKey util.Ed25519PublicKey `yaml:"public-key"`
	TokenHash util.HexBytes         `yaml:"token-hash"`
}

type NetworkConfig struct {
	Name string   `yaml:"name"`
	IPs  []string `yaml:"ips"`
	Host string   `yaml:"host"`
}

var tcPath string
var cfg Config
var tc TmpConfig

func main() {
	flag.StringVar(&tcPath, "tmp-config", "", "path to tmp config file")
	flag.Parse()

	raw, err := ioutil.ReadFile(tcPath)
	if err != nil {
		log.Fatalf("config read: %s", err)
	}
	err = yaml.Unmarshal(raw, &tc)
	if err != nil {
		log.Fatalf("config unmarshal: %s", err)
	}

	raw, err = ioutil.ReadFile(tc.ConfigPath)
	if err != nil {
		log.Fatalf("config read: %s", err)
	}
	err = yaml.Unmarshal(raw, &cfg)
	if err != nil {
		log.Fatalf("config unmarshal: %s", err)
	}

	creds, err := credentials.NewClientTLSFromFile(cfg.TLSCertPath, "")
	if err != nil {
		log.Fatalf("client tls: %s", err)
	}

	conn, err := grpc.Dial(cfg.Server, grpc.WithTimeout(5*time.Second), grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("dial: %s", err)
	}
	networks := map[string]string{}
	for cnn, nc := range tc.Networks {
		networks[cnn] = nc.Name
	}
	cl := api.NewCentralSourceClient(conn)
	_, err = cl.AddToken(context.Background(), &api.AddTokenQ{
		CentralToken: cfg.CentralToken,
		Overwrite:    tc.Overwrite,
		Hash:         tc.TokenHash,
		Name:         tc.Name,
		Networks:     networks,
		CanPull:      true,
	})
	if err != nil {
		log.Fatalf("add token: %s", err)
	}
	for cnn, nc := range tc.Networks {
		allowedIPs := make([]net.IPNet, len(nc.IPs))
		for i, raw := range nc.IPs {
			_, allowedIP, err := net.ParseCIDR(raw)
			if err != nil {
				log.Fatalf("parse ip %d: %s", i, err)
			}
			allowedIPs[i] = *allowedIP
		}

		_, err := cl.Push(context.Background(), &api.PushQ{
			CentralToken: cfg.CentralToken,
			Cnn:          cnn,
			PeerName:     nc.Name,
			Peer: &api.CentralPeer{
				Host:       nc.Host,
				AllowedIPs: cs.FromIPNets(allowedIPs),
				PublicKey:  &api.PublicKey{Raw: tc.PublicKey},
			},
		})
		if err != nil {
			log.Fatalf("push net %s: %s", cnn, err)
		}
	}
}
