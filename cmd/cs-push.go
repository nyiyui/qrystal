package main

import (
	"context"
	"encoding/base64"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/nyiyui/qanms/cs"
	"github.com/nyiyui/qanms/node/api"
	"github.com/nyiyui/qanms/util"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       string           `yaml:"server"`
	CentralToken util.Base64Bytes `yaml:"token"`
}

var configPath string
var cnn string
var pn string
var ph string
var pubKeyRaw string
var config Config

func main() {
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.StringVar(&cnn, "cnn", "", "network name")
	flag.StringVar(&pn, "pn", "", "peer name")
	flag.StringVar(&ph, "ph", "", "peer host")
	flag.StringVar(&pubKeyRaw, "pubKey", "", "peer public key")
	flag.Parse()

	pubKey, err := base64.StdEncoding.DecodeString(pubKeyRaw)
	if err != nil {
		log.Fatalf("decode public key: %s", err)
	}

	allowedIPs := make([]net.IPNet, len(flag.Args()))
	for i, raw := range flag.Args() {
		_, allowedIP, err := net.ParseCIDR(raw)
		if err != nil {
			log.Fatalf("parse ip %d: %s", i, err)
		}
		allowedIPs[i] = *allowedIP
	}

	raw, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("config read: %s", err)
	}
	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		log.Fatalf("config unmarshal: %s", err)
	}

	conn, err := grpc.Dial(config.Server, grpc.WithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("dial: %s", err)
	}
	cl := api.NewCentralSourceClient(conn)
	q := &api.PushQ{
		CentralToken: config.CentralToken,
		Cnn:          cnn,
		PeerName:     pn,
		Peer: &api.CentralPeer{
			Host:       ph,
			AllowedIPs: cs.FromIPNets(allowedIPs),
			PublicKey:  &api.PublicKey{Raw: pubKey},
		},
	}
	s, err := cl.Push(context.Background(), q)
	if err != nil {
		log.Fatalf("push: %s", err)
	}
	log.Printf("ok: %#v", s)
}
