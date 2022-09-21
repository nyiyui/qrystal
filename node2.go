package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nyiyui/qanms/node"
	"github.com/nyiyui/qanms/node/api"
	"gopkg.in/yaml.v3"
)

var configPath string
var mioPort uint

type config struct {
	ResyncEvery time.Duration      `yaml:"resync-every"`
	PrivKey     string             `yaml:"private-key"`
	Addr        string             `yaml:addr"`
	Central     node.CentralConfig `yaml:"central"`
}

func main() {
	flag.StringVar(&configPath, "config", "./config.yml", "config path")
	flag.UintVar(&mioPort, "mio-port", 0, "mio port")
	flag.Parse()

	var c config
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		panic(err)
	}

	if c.PrivKey[0] != 'R' {
		log.Fatal("not a private key")
	}
	privKey, err := base64.StdEncoding.DecodeString(c.PrivKey[2:])
	if err != nil {
		log.Fatal(err)
	}
	privKey2 := ed25519.NewKeyFromSeed([]byte(privKey))

	for nn, cn := range c.Central.Networks {
		for pn, peer := range cn.Peers {
			if peer.PublicKeyInput[0] != 'U' {
				log.Fatalf("network %s / peer %s: not a public key")
			}
			peer.PublicKey, err = base64.StdEncoding.DecodeString(peer.PublicKeyInput[2:])
			if err != nil {
				log.Fatalf("network %s / peer %s: %s", nn, pn, err)
			}
		}
	}

	c.Central.DialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	n, err := node.NewNode(node.NodeConfig{
		PrivKey: privKey2,
		CC:      c.Central,
		MioPort: uint16(mioPort),
	})
	if err != nil {
		panic(err)
	}

	go func() {
		server := grpc.NewServer()
		api.RegisterNodeServer(server, n)
		lis, err := net.Listen("tcp", c.Addr)
		if err != nil {
			log.Fatal(err)
		}
		err = server.Serve(lis)
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		ticker := time.Tick(c.ResyncEvery)
		var syncIndex int
		log.Printf("sync %d on %s", syncIndex, time.Now())
		syncRes, err := n.Sync(context.Background())
		if err != nil {
			log.Printf("sync %d error: %s", syncIndex, err)
		}
		log.Printf("sync %d res: %s", syncIndex, syncRes)
		syncIndex++
		select {
		case now := <-ticker:
			log.Printf("sync %d on %s", syncIndex, now)
			syncRes, err := n.Sync(context.Background())
			if err != nil {
				log.Printf("sync %d error: %s", syncIndex, err)
			}
			log.Printf("sync %d res: %s", syncIndex, syncRes)
			syncIndex++
			log.Fatal("end")
		}
	}()
	select {}
}
