package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/nyiyui/qrystal/node"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"gopkg.in/yaml.v3"
)

type config struct {
	PrivKey string             `yaml:"private-key"`
	Server  *serverConfig      `yaml:"server"`
	Central node.CentralConfig `yaml:"central"`
	CS      *csConfig          `yaml:"cs"`
	Azusa   *azusaConfig       `yaml:"azusa"`
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
	Host  string `yaml:"host"`
	Token string `yaml:"token"`
}

func main() {
	log.SetPrefix("node: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	util.ShowCurrent()

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

	clientCreds := credentials.NewTLS(nil)

	var serverCreds credentials.TransportCredentials
	if c.Server != nil {
		s := c.Server
		serverCreds, err = credentials.NewServerTLSFromFile(s.TLSCertPath, s.TLSKeyPath)
		if err != nil {
			log.Fatalf("client tls: %s", err)
		}
	}

	c.Central.DialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(clientCreds),
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
		CSCreds:  clientCreds,
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
		go func() {
			err := n.ListenCS()
			if err != nil {
				log.Printf("listen: %s", err)
			}
		}()
	}
	select {}
}
