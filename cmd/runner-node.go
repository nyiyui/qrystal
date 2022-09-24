package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nyiyui/qanms/node"
	"github.com/nyiyui/qanms/node/api"
	"gopkg.in/yaml.v3"
)

type config struct {
	ResyncEvery time.Duration      `yaml:"resync-every"`
	PrivKey     string             `yaml:"private-key"`
	Addr        string             `yaml:"addr"`
	Central     node.CentralConfig `yaml:"central"`
	CS          csConfig           `yaml:"cs"`
}

type csConfig struct {
	Host  string `yaml:"host"`
	Token string `yaml:"token"`
}

func main() {
	log.SetPrefix("node: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	var c config
	data, err := ioutil.ReadFile(os.Getenv("CONFIG_PATH"))
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

	c.Central.DialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	mioPort, err := strconv.ParseUint(os.Getenv("MIO_PORT"), 10, 16)
	if err != nil {
		log.Fatalf("parse MIO_PORT: %s", err)
	}
	mioToken, err := base64.StdEncoding.DecodeString(os.Getenv("MIO_TOKEN"))
	if err != nil {
		log.Fatalf("parse MIO_TOKEN: %s", err)
	}
	n, err := node.NewNode(node.NodeConfig{
		PrivKey:  privKey2,
		CC:       c.Central,
		MioPort:  uint16(mioPort),
		MioToken: mioToken,
		CSHost:   c.CS.Host,
		CSToken:  c.CS.Token,
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
	/*
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
			}
		}()
	*/
	go func() {
		err := n.ListenCS()
		if err != nil {
			log.Printf("listen: %s", err)
		}
	}()
	/*
			log.Printf("CSに接続します。 %s", c.CS.Host)
			conn, err := grpc.Dial(c.CS.Host, grpc.WithTimeout(5*time.Second),
				grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Fatalf("dial: %s", err)
			}
			cl := api.NewCentralSourceClient(conn)
			_, err = cl.Ping(context.Background(), &api.PingQS{})
			if err != nil {
				log.Fatalf("ping: %s", err)
			}
			log.Printf("CSに接続しました。 %s", c.CS.Host)
			call, err := cl.Pull(context.Background(), &api.PullQ{
				CentralToken: c.CS.Token,
			})
			if err != nil {
				log.Fatalf("pull init: %s", err)
			}
			for {
				s, err := call.Recv()
				if err != nil {
					log.Fatalf("dial: %s", err)
				}
				log.Print("pull received")
				cc, err := node.NewCCFromAPI(s.Cc)
				if err != nil {
					log.Fatalf("convert: %s", err)
				}
				n.ReplaceCC(cc)
			}
		}()
	*/
	select {}
}
