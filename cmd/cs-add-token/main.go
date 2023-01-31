package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type AddTokenQ struct {
	Overwrite bool          `json:"overwrite"`
	Name      string        `json:"name"`
	Hash      util.HexBytes `json:"hash"`
	CanPull   *struct {
		Networks map[string]string `json:"networks"`
	} `json:"canPull"`
	CanPush *struct {
		Networks map[string]Peer `json:"networks"`
	} `json:"canPush"`
}

type Peer struct {
	Name          string   `json:"name"`
	CanSeeElement []string `json:"canSeeElement"`
}

var cfgServer string
var cfgCT string
var certPath string

func main() {
	flag.StringVar(&cfgServer, "server", "", "server address")
	flag.StringVar(&cfgCT, "token", "", "central token")
	flag.StringVar(&certPath, "cert", "", "path to server cert")
	flag.Parse()

	var q AddTokenQ
	err := json.NewDecoder(os.Stdin).Decode(&q)
	if err != nil {
		log.Fatalf("unmarshal config: %s", err)
	}

	creds, err := credentials.NewClientTLSFromFile(certPath, "")
	if err != nil {
		log.Fatalf("load cert: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, cfgServer, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("dial: %s", err)
	}
	cl := api.NewCentralSourceClient(conn)
	q2 := api.AddTokenQ{
		CentralToken: cfgCT,
		Overwrite:    q.Overwrite,
		Hash:         q.Hash,
		Name:         q.Name,
	}
	if q.CanPull != nil {
		q2.CanPull = true
		q2.Networks = q.CanPull.Networks
	}
	if q.CanPush != nil {
		networks := map[string]string{}
		ncse := map[string]*api.LString{}
		for key, peer := range q.CanPush.Networks {
			networks[key] = peer.Name
			ncse[key] = &api.LString{Inner: peer.CanSeeElement}
		}
		q2.CanPush = &api.CanPush{
			Networks: networks,
		}
		q2.CanPushNetworksCanSeeElement = ncse
	}
	_, err = cl.AddToken(context.Background(), &q2)
	if err != nil {
		log.Fatalf("add token: %s", err)
	}
}
