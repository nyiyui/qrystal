package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"github.com/nyiyui/qrystal/cs"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       string `yaml:"server"`
	CentralToken string `yaml:"token"`
}

type TmpConfig struct {
	Overwrite bool                     `yaml:"overwrite"`
	Name      string                   `yaml:"name"`
	Networks  map[string]NetworkConfig `yaml:"networks"`

	TokenHash string `yaml:"tokenHash"`
}

type NetworkConfig struct {
	Name    string   `yaml:"name"`
	IPs     []string `yaml:"ips"`
	Host    string   `yaml:"host"`
	CanPush bool     `yaml:"canPush"`
	CanSee  []string `yaml:"canSee"`
}

var tcPath string
var cfg Config
var cfgServer string
var tc TmpConfig
var certPath string

func main() {
	flag.StringVar(&cfgServer, "server", "", "server address")
	ctRaw := flag.String("token", "", "central token")
	flag.StringVar(&tcPath, "tmp-config", "", "path to tmp config file")
	flag.StringVar(&certPath, "cert", "", "path to server cert")
	flag.Parse()

	ct, err := util.ParseToken(*ctRaw)
	if err != nil {
		log.Fatalf("parse token: %s", err)
	}

	raw, err := os.ReadFile(tcPath)
	if err != nil {
		log.Fatalf("config read: %s", err)
	}
	err = yaml.Unmarshal(raw, &tc)
	if err != nil {
		log.Fatalf("config unmarshal: %s", err)
	}

	cfg.Server = cfgServer
	cfg.CentralToken = ct.String()

	creds, err := credentials.NewClientTLSFromFile(certPath, "")
	if err != nil {
		log.Fatalf("load cert: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, cfg.Server, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("dial: %s", err)
	}
	canPushNets := map[string]string{}
	networks := map[string]string{}
	for cnn, nc := range tc.Networks {
		networks[cnn] = nc.Name
		if nc.CanPush {
			canPushNets[cnn] = nc.Name
		}
	}
	cl := api.NewCentralSourceClient(conn)

	_, err = cl.AddToken(context.Background(), &api.AddTokenQ{
		CentralToken: cfg.CentralToken,
		Overwrite:    tc.Overwrite,
		Hash:         tc.TokenHash,
		Name:         tc.Name,
		Networks:     networks,
		CanPull:      true,
		CanPush: &api.CanPush{
			Networks: canPushNets,
		},
	})
	if err != nil {
		log.Fatalf("add token: %s", err)
	}
	for cnn, nc := range tc.Networks {
		allowedIPs := make([]net.IPNet, len(nc.IPs))
		for i, raw := range nc.IPs {
			allowedIPs[i], err = util.ParseCIDR(raw)
			if err != nil {
				log.Fatalf("parse ip %d: %s", i, err)
			}
		}

		_, err := cl.Push(context.Background(), &api.PushQ{
			CentralToken: cfg.CentralToken,
			Cnn:          cnn,
			PeerName:     nc.Name,
			Peer: &api.CentralPeer{
				Host:       nc.Host,
				AllowedIPs: cs.FromIPNets(allowedIPs),
				CanSee:     &api.CanSee{Only: nc.CanSee},
			},
		})
		if err != nil {
			log.Fatalf("push net %s: %s", cnn, err)
		}
	}
}
