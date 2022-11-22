package node

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type NodeConfig struct {
	PrivKey  ed25519.PrivateKey
	CC       central.Config
	MioPort  uint16
	MioToken []byte
	CS       []CSConfig
}

func NewNode(cfg NodeConfig) (*Node, error) {
	for cnn, cn := range cfg.CC.Networks {
		cn.Name = cnn
		for pn, peer := range cn.Peers {
			peer.Name = pn
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
	cc           central.Config
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

func (s *Node) addRandomToken(cn *central.Network, clientName string) (token []byte, err error) {
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
		cnn:  cn.Name,
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
		return fmt.Errorf("solve chall: %w", err)
	}
	err = state.verifyChall(state.cn.Name, state.you.Name)
	if err != nil {
		return fmt.Errorf("verify chall: %w", err)
	}
	util.S.Debugf("net %s peer %s: generating token", state.cn.Name, state.you.Name)
	err = func() error {
		state.cn.Lock.Lock()
		defer state.cn.Lock.Unlock()
		token, err := s.addRandomToken(state.cn, state.you.Name)
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
	sc, ok := s.readToken(q.Token)
	if !ok {
		return nil, errors.New("unknown token")
	}
	cnn := sc.cnn
	s.Kiriyama.SetPeer(cnn, sc.name, "交換中")
	cn, ok := s.cc.Networks[cnn]
	if !ok {
		return nil, errors.New("unknown network")
	}

	var you *central.Peer
	you = cn.Peers[sc.name]
	you.LSALock.Lock()
	defer you.LSALock.Unlock()
	if time.Since(you.LSA) < 1*time.Second {
		return nil, errors.New("attempted to sync too recently")
	}

	var myPubKey wgtypes.Key
	err = func() error {
		you.Lock.Lock()
		defer you.Lock.Unlock()
		yourPubKey, err := wgtypes.NewKey(q.PubKey)
		if err != nil {
			return errors.New("invalid public key")
		}
		you.PubKey = &yourPubKey
		yourPSK, err := wgtypes.NewKey(q.Psk)
		if err != nil {
			return errors.New("invalid psk")
		}
		you.PSK = &yourPSK

		util.S.Debugf("net %s peer %s: generating", cnn, you.Name)
		err = ensureWGPrivKey(cn)
		if err != nil {
			return errors.New("private key generation failed")
		}
		myPubKey = cn.MyPrivKey.PublicKey()

		you.Accessible = true
		return nil
	}()
	if err != nil {
		return nil, err
	}

	util.S.Debugf("net %s peer %s: configuring network", cnn, you.Name)
	// TODO: consider running this in a goroutine or something
	err = s.configNetwork(cn)
	if err != nil {
		util.S.Errorf("configuration of net %s failed (iniiated by peer %s):\n%s", cn.Name, you.Name, err)
		return nil, errors.New("configuration failed")
	}
	s.Kiriyama.SetPeer(cnn, sc.name, "交換OK")

	return &api.XchS{
		PubKey: myPubKey[:],
	}, nil
}

func (s *Node) Ping(context.Context, *api.PingQS) (*api.PingQS, error) {
	log.Print("server: pinged")
	return &api.PingQS{}, nil
}
