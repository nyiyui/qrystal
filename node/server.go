package node

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/node/api"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"google.golang.org/grpc/credentials"
)

type NodeConfig struct {
	PrivKey  ed25519.PrivateKey
	CC       CentralConfig
	MioPort  uint16
	MioToken []byte
	CSHost   string
	CSToken  string
	CSCreds  credentials.TransportCredentials
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
		csHost:  cfg.CSHost,
		csToken: cfg.CSToken,
		csCreds: cfg.CSCreds,
	}
	return node, nil
}

type Node struct {
	api.UnimplementedNodeServer
	ccLock       sync.RWMutex
	cc           CentralConfig
	coordPrivKey ed25519.PrivateKey
	csHost       string
	csToken      string
	csCreds      credentials.TransportCredentials

	state       serverState
	serversLock sync.RWMutex
	servers     map[networkPeerPair]*clientServer

	mio *mioHandle
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
	err := state.authOthers()
	if err != nil {
		return fmt.Errorf("authenticating you: %w", err)
	}
	err = state.authMine(state.cn.name, state.you.name)
	if err != nil {
		return fmt.Errorf("authenticating me: %w", err)
	}
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
	/*
		var cn CentralNetwork
		var you CentralPeer
		{
			sq1Raw, err := conn.Recv()
			if err != nil {
				return err
			}
			sq1 := sq1Raw.Sq.(*api.AuthSQ_Q).Q
			var ok bool
			cn, ok = s.cc.Networks[sq1.Network]
			if !ok {
				return errors.New("unknown network")
			}
			you, ok = cn.Peers[sq1.Me]
			if !ok {
				return errors.New("unknown you")
			}

			var added []byte
			var challResp []byte
			{
				added, err = readRand(32)
				if err != nil {
					return errors.New("generating challenge added failed")
				}

				signThis := make([]byte, 64)
				copy(signThis, sq1.Chall)
				signThis = append(signThis, added...)
				challResp = ed25519.Sign(s.coordPrivKey, signThis)
			}

			sq2 := api.AuthS{
				ChallResp:  challResp,
				ChallAdded: added,
			}
			err = conn.Send(&api.AuthSQ{Sq: &api.AuthSQ_S{S: &sq2}})
			if err != nil {
				return err
			}
		}

		{
			chall, err := readRand(32)
			if err != nil {
				return errors.New("gen chall failed")
			}
			sq3 := api.AuthQ{
				Chall: chall,
			}
			err = conn.Send(&api.AuthSQ{Sq: &api.AuthSQ_Q{Q: &sq3}})
			if err != nil {
				return err
			}
			sq4Raw, err := conn.Recv()
			if err != nil {
				return err
			}
			sq4 := sq4Raw.Sq.(*api.AuthSQ_S).S
			signed := make([]byte, 64)
			copy(signed, chall)
			signed = append(signed, sq4.ChallAdded...)
			ok := ed25519.Verify(you.PublicKey, signed, sq4.ChallResp)
			if !ok {
				return errors.New("signature verification failed")
			}
		}
	*/
}

func (s *Node) readToken(token []byte) (sc serverClient, ok bool) {
	s.state.lock.RLock()
	defer s.state.lock.RUnlock()
	sc, ok = s.state.tokenSecrets[string(token)]
	return sc, ok
}

func (s *Node) configNetwork(cn *CentralNetwork) (err error) {
	config, err := s.convertNetwork(cn)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	me := cn.Peers[cn.Me]
	q := mio.ConfigureDeviceQ{
		Name:    cn.name,
		Config:  config,
		Address: ToIPNets(me.AllowedIPs),
		// TODO: fix to use my IPs
	}
	err = s.mio.ConfigureDevice(q)
	if err != nil {
		return fmt.Errorf("mio: %w", err)
	}
	return nil
}

func (s *Node) convertNetwork(cn *CentralNetwork) (config *wgtypes.Config, err error) {
	cn.lock.RLock()
	defer cn.lock.RUnlock()
	configs := make([]wgtypes.PeerConfig, 0, len(cn.Peers))
	for pn, peer := range cn.Peers {
		if !peer.accessible {
			continue
		}
		config, err := s.convertPeer(cn, peer)
		if err != nil {
			return nil, fmt.Errorf("peer %s: %w", pn, err)
		}
		configs = append(configs, *config)
	}
	config = &wgtypes.Config{
		PrivateKey:   cn.myPrivKey,
		ListenPort:   &cn.ListenPort,
		ReplacePeers: true,
		Peers:        configs,
	}
	return config, nil
}

func (s *Node) convertPeer(cn *CentralNetwork, peer *CentralPeer) (config *wgtypes.PeerConfig, err error) {
	host, err := net.ResolveUDPAddr("udp", peer.Host)
	if err != nil {
		return nil, fmt.Errorf("resolving peer host %s failed", peer.Host)
	}

	if peer.pubKey == nil {
		panic(fmt.Sprintf("net %#v peer %#v pubKey is nil", cn, peer))
	}

	return &wgtypes.PeerConfig{
		PublicKey:                   *peer.pubKey,
		Remove:                      false,
		UpdateOnly:                  false,
		PresharedKey:                peer.psk,
		Endpoint:                    host,
		PersistentKeepaliveInterval: &cn.Keepalive,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  ToIPNets(peer.AllowedIPs),
	}, nil

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
	you.lock.Lock()
	defer you.lock.Unlock()
	yourPubKey, err := wgtypes.NewKey(q.PubKey)
	if err != nil {
		return nil, errors.New("invalid public key")
	}
	you.pubKey = &yourPubKey
	yourPSK, err := wgtypes.NewKey(q.Psk)
	if err != nil {
		return nil, errors.New("invalid psk")
	}
	you.psk = &yourPSK
	log.Println("SET2 PSK:", you, yourPSK)

	err = ensureWGPrivKey(cn)
	if err != nil {
		return nil, errors.New("private key generation failed")
	}
	myPubKey := cn.myPrivKey.PublicKey()

	you.accessible = true

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
	return &api.PingQS{}, nil
}
