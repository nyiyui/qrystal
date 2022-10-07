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
	TLSCertPath string             `yaml:"tls-cert-path"`
	TLSKeyPath  string             `yaml:"tls-key-path"`
	PrivKey     string             `yaml:"private-key"`
	Addr        string             `yaml:"addr"`
	Central     node.CentralConfig `yaml:"central"`
	CS          csConfig           `yaml:"cs"`
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
	util.ShowCurrent()
	log.SetPrefix("node: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

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

	clientCreds, err := credentials.NewClientTLSFromFile(c.TLSCertPath, "")
	if err != nil {
		log.Fatalf("client tls: %s", err)
	}

	serverCreds, err := credentials.NewServerTLSFromFile(c.TLSCertPath, c.TLSKeyPath)
	if err != nil {
		log.Fatalf("client tls: %s", err)
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

	go func() {
		server := grpc.NewServer(grpc.Creds(serverCreds))
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
		err := n.ListenCS()
		if err != nil {
			log.Printf("listen: %s", err)
		}
	}()
	select {}
}
