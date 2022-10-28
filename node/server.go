package node

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/node/api"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type NodeConfig struct {
	PrivKey  ed25519.PrivateKey
	CC       CentralConfig
	MioPort  uint16
	MioToken []byte
	CS       []CSConfig
}

func NewNode(cfg NodeConfig) (*Node, error) {
	for cnn, cn := range cfg.CC.Networks {
		cn.name = cnn
		for pn, peer := range cn.Peers {
			peer.name = pn
			if pn == cn.Me {
				// check pub/priv key pair
				privKey := cfg.PrivKey
				derived := privKey.Public().(ed25519.PublicKey)
				pubKey := ed25519.PublicKey(peer.PublicKey)
				if !pubKey.Equal(derived) {
					return nil, fmt.Errorf("my public and private keys don't match:\npublic: %s\nprivate: %s\npublic from private: %s ", pubKey, privKey, derived)
				}
			}
		}
	}

	mh, err := newMio(cfg.MioPort, cfg.MioToken)
	if err != nil {
		return nil, fmt.Errorf("new mio: %w", err)
	}

	node := &Node{
		cc:           cfg.CC,
		coordPrivKey: cfg.PrivKey,

		state: serverState{
			tokenSecrets: map[string]serverClient{},
		},
		servers: map[networkPeerPair]*clientServer{},
		mio:     mh,
		cs:      cfg.CS,
		csNets:  map[string]int{},

		azusa: new(azusa),
		csCls: make([]api.CentralSourceClient, len(cfg.CS)),
	}
	node.Kiriyama = newKiriyama(node)
	return node, nil
}

type Node struct {
	api.UnimplementedNodeServer
	ccLock       sync.RWMutex
	cc           CentralConfig
	coordPrivKey ed25519.PrivateKey
	cs           []CSConfig
	csNets       map[string]int

	state       serverState
	serversLock sync.RWMutex
	servers     map[networkPeerPair]*clientServer

	mio *mioHandle

	azusa *azusa
	csCls []api.CentralSourceClient

	Kiriyama *Kiriyama
}

var _ api.NodeServer = (*Node)(nil)

type serverState struct {
	lock         sync.RWMutex
	tokenSecrets map[string]serverClient
}

type serverClient struct {
	name string
	cnn  string
}

// NOTES: Authn/z
//     last AuthS provides random token by server for client to use in authn/zed calls

func (s *Node) addRandomToken(cn *CentralNetwork, clientName string) (token []byte, err error) {
	token, err = readRand(65)
	if err != nil {
		return nil, err
	}
	_, ok := cn.Peers[clientName]
	if !ok {
		return nil, errors.New("client not found")
	}
	s.state.lock.Lock()
	defer s.state.lock.Unlock()
	s.state.tokenSecrets[string(token)] = serverClient{
		name: clientName,
		cnn:  cn.name,
	}
	return token, nil
}

func (s *Node) Auth(conn api.Node_AuthServer) error {
	s.ccLock.RLock()
	defer s.ccLock.RUnlock()
	state := authState{
		coordPrivKey: s.coordPrivKey,
		conn:         conn,
		cc:           s.cc,
	}
	err := state.solveChall()
	if err != nil {
		log.Printf("solve chall failed: %s", err)
		return fmt.Errorf("solve chall: %w", err)
	}
	err = state.verifyChall(state.cn.name, state.you.name)
	if err != nil {
		log.Printf("verify chall failed: %s", err)
		return fmt.Errorf("verify chall: %w", err)
	}
	log.Printf("net %s peer %s: generating token", state.cn.name, state.you.name)
	err = func() error {
		state.cn.lock.Lock()
		defer state.cn.lock.Unlock()
		token, err := s.addRandomToken(state.cn, state.you.name)
		log.Printf("you: %#v", state.you)
		log.Printf("cn: %#v", state.cn)
		if err != nil {
			return fmt.Errorf("generating token failed: %w", err)
		}
		sq5 := api.AuthToken{
			Token: token,
		}
		sq5Raw := api.AuthSQ{Sq: &api.AuthSQ_Token{Token: &sq5}}
		err = conn.Send(&sq5Raw)
		if err != nil {
			return err
		}
		return nil
	}()
	return err
}

func (s *Node) readToken(token []byte) (sc serverClient, ok bool) {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()
	sc, ok = s.state.tokenSecrets[string(token)]
	return sc, ok
}

func (s *Node) Xch(ctx context.Context, q *api.XchQ) (r *api.XchS, err error) {
	s.ccLock.RLock()
	defer s.ccLock.RUnlock()
	log.Printf("==XCH %#v", q)
	sc, ok := s.readToken(q.Token)
	if !ok {
		return nil, errors.New("unknown token")
	}
	cnn := sc.cnn
	cn, ok := s.cc.Networks[cnn]
	if !ok {
		return nil, errors.New("unknown network")
	}

	var you *CentralPeer
	you = cn.Peers[sc.name]
	you.lsaLock.Lock()
	defer you.lsaLock.Unlock()
	if time.Since(you.lsa) < 1*time.Second {
		return nil, errors.New("attempted to sync too recently")
	}

	var myPubKey wgtypes.Key
	err = func() error {
		you.lock.Lock()
		log.Printf("LOCKYOU net %s peer %s", cnn, sc.name)
		defer you.lock.Unlock()
		yourPubKey, err := wgtypes.NewKey(q.PubKey)
		if err != nil {
			return errors.New("invalid public key")
		}
		you.pubKey = &yourPubKey
		yourPSK, err := wgtypes.NewKey(q.Psk)
		if err != nil {
			return errors.New("invalid psk")
		}
		you.psk = &yourPSK
		log.Println("SET2 PSK:", you, yourPSK)

		log.Printf("net %s peer %s: generating", cnn, you.name)
		err = ensureWGPrivKey(cn)
		if err != nil {
			return errors.New("private key generation failed")
		}
		myPubKey = cn.myPrivKey.PublicKey()

		you.accessible = true
		return nil
	}()
	if err != nil {
		return nil, err
	}

	log.Printf("net %s peer %s: configuring network", cnn, you.name)
	// TODO: consider running this in a goroutine or something
	err = s.configNetwork(cn)
	if err != nil {
		log.Printf("configuration of net %s failed (iniiated by peer %s):\n%s", cn.name, you.name, err)
		return nil, errors.New("configuration failed")
	}

	return &api.XchS{
		PubKey: myPubKey[:],
	}, nil
}

func (s *Node) Ping(context.Context, *api.PingQS) (*api.PingQS, error) {
	log.Print("server: pinged")
	return &api.PingQS{}, nil
}
