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
	"github.com/nyiyui/qrystal/node/verify"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type NodeConfig struct {
	PrivKey  ed25519.PrivateKey
	CC       central.Config
	MioAddr  string
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

	mh, err := newMio(cfg.MioAddr, cfg.MioToken)
	if err != nil {
		return nil, fmt.Errorf("new mio: %w", err)
	}

	node := &Node{
		cc:           cfg.CC,
		coordPrivKey: cfg.PrivKey,

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

	serversLock sync.RWMutex
	servers     map[networkPeerPair]*clientServer

	mio *mioHandle

	azusa *azusa
	csCls []api.CentralSourceClient

	Kiriyama *Kiriyama
}

var _ api.NodeServer = (*Node)(nil)

func (s *Node) Xch(ctx context.Context, q *api.XchQ) (r *api.XchS, err error) {
	s.ccLock.RLock()
	defer s.ccLock.RUnlock()

	cnn := q.Cnn
	cn, ok := s.cc.Networks[cnn]
	if !ok {
		return nil, errors.New("unknown network")
	}

	var you *central.Peer
	you = cn.Peers[q.Peer]

	err = verify.VerifyXchQ(ed25519.PublicKey(you.PublicKey), q)
	if err != nil {
		util.S.Warnf("VerifyXchQ (signed by %s): %s", q.Peer, err)
		return nil, errors.New("q verification failed")
	}

	you.Internal.LSALock.Lock()
	defer you.Internal.LSALock.Unlock()
	if time.Since(you.Internal.LSA) < verify.Grave {
		return nil, errors.New("attempted to sync too recently")
	}

	s.Kiriyama.SetPeer(cnn, q.Peer, "交換中")

	var myPubKey wgtypes.Key
	err = func() error {
		you.Internal.Lock.Lock()
		defer you.Internal.Lock.Unlock()
		yourPubKey, err := wgtypes.NewKey(q.PubKey)
		if err != nil {
			return errors.New("invalid public key")
		}
		you.Internal.PubKey = &yourPubKey
		yourPSK, err := wgtypes.NewKey(q.Psk)
		if err != nil {
			return errors.New("invalid psk")
		}
		you.Internal.PSK = &yourPSK

		util.S.Debugf("net %s peer %s: generating", cnn, you.Name)
		err = ensureWGPrivKey(cn)
		if err != nil {
			return errors.New("private key generation failed")
		}
		myPubKey = cn.MyPrivKey.PublicKey()

		you.Internal.Accessible = true
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
	s.Kiriyama.SetPeer(cnn, q.Peer, "交換OK")

	s2 := &api.XchS{
		PubKey: myPubKey[:],
		Ts:     time.Now().Format(time.RFC3339),
		Sig:    nil,
	}
	err = verify.SignXchS(s.coordPrivKey, s2)
	if err != nil {
		return nil, err
	}
	return s2, nil
}

func (s *Node) Ping(context.Context, *api.PingQS) (*api.PingQS, error) {
	log.Print("server: pinged")
	return &api.PingQS{}, nil
}
