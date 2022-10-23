package node

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gopkg.in/yaml.v3"
)

type SyncRes struct {
	netStatus map[string]SyncNetRes
}

func (r *SyncRes) allOK() bool {
	for _, netRes := range r.netStatus {
		if !netRes.allOK() {
			return false
		}
	}
	return true
}

func (r *SyncRes) String() string {
	b := new(strings.Builder)
	for nn, ns := range r.netStatus {
		fmt.Fprintf(b, "net %s:\n%s\n", nn, &ns)
	}
	return b.String()
}

type SyncNetRes struct {
	err        error
	peerStatus map[string]SyncPeerRes
}

func (r *SyncNetRes) allOK() bool {
	if r.err != nil {
		return false
	}
	for _, peerRes := range r.peerStatus {
		if peerRes.err != nil {
			return false
		}
		if peerRes.forwardErr != nil {
			return false
		}
	}
	return true
}

func (r *SyncNetRes) String() string {
	b := new(strings.Builder)
	if r.err != nil {
		fmt.Fprintf(b, "\terr: %s\n", r.err)
	} else {
		fmt.Fprint(b, "\tno error\n")
	}
	for pn, ps := range r.peerStatus {
		fmt.Fprintf(b, "\tpeer %s: %s\n", pn, &ps)
	}
	return b.String()
}

type SyncPeerRes struct {
	skip       bool
	err        error
	forwardErr error
}

func (r *SyncPeerRes) String() string {
	b := new(strings.Builder)
	if r.skip {
		fmt.Fprintf(b, "\t\tskip\n")
	} else {
		fmt.Fprintf(b, "\t\txch     err: %s\n", r.err)
		fmt.Fprintf(b, "\t\tforward err: %s\n", r.err)
	}
	return b.String()
}

func (c *Node) Sync(ctx context.Context, xch bool) (*SyncRes, error) {
	res := SyncRes{
		netStatus: map[string]SyncNetRes{},
	}
	c.ccLock.RLock()
	defer c.ccLock.RUnlock()
	ccy, _ := yaml.Marshal(c.cc)
	util.S.Infof("cc: %s", ccy)
	for cnn := range c.cc.Networks {
		log.Printf("===SYNCING net %s", cnn)
		netRes, err := c.syncNetwork(ctx, cnn, xch)
		if netRes == nil {
			netRes = &SyncNetRes{}
		}
		netRes.err = err
		res.netStatus[cnn] = *netRes
	}
	return &res, nil
}

func (c *Node) syncNetwork(ctx context.Context, cnn string, xch bool) (*SyncNetRes, error) {
	cn := c.cc.Networks[cnn]
	err := ensureWGPrivKey(cn)
	if err != nil {
		return nil, errors.New("private key generation failed")
	}

	res := SyncNetRes{
		peerStatus: map[string]SyncPeerRes{},
	}
	pns := make([]string, 0)
	{
		var pnsLock sync.Mutex
		var wg sync.WaitGroup
		for pn := range cn.Peers {
			if xch {
				if pn == cn.Me {
					continue
				}
				wg.Add(1)
				go func(pn string) {
					defer wg.Done()
					err := c.ensureClient(ctx, cnn, pn)
					if err != nil {
						res.peerStatus[pn] = SyncPeerRes{
							err: err,
						}
						return
					}
					log.Printf("net %s peer %s syncing", cn.name, pn)
					ps := c.xchPeer(ctx, cnn, pn)
					log.Printf("net %s peer %s synced: %s", cn.name, pn, &ps)
					res.peerStatus[pn] = ps
					if ps.err == nil && !ps.skip {
						pnsLock.Lock()
						defer pnsLock.Unlock()
						pns = append(pns, pn)
					}
				}(pn)
			}
		}
		wg.Wait()
		pnsLock.Lock()
	}

	if xch {
		log.Printf("net %s peers %s advertising forwarding capability", cn.name, pns)
		csI, err := c.getCSForNet(cnn)
		if err != nil {
			return nil, fmt.Errorf("getCSForNet: %w", err)
		}
		_, err = c.csCls[csI].CanForward(ctx, &api.CanForwardQ{
			CentralToken:   c.cs[csI].Token,
			Network:        cnn,
			ForwardeePeers: pns,
		})
		if err != nil {
			return nil, fmt.Errorf("CanForward: %w", err)
		}
	}
	err = c.configNetwork(cn)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Node) xchPeer(ctx context.Context, cnn string, pn string) (res SyncPeerRes) {
	cn := c.cc.Networks[cnn]
	skip := func() bool {
		peer := cn.Peers[pn]
		peer.lock.RLock()
		log.Printf("LOCK net %s peer %s", cnn, pn)
		defer peer.lock.RUnlock()
		log.Printf("net %s peer %s: %#v", cnn, pn, peer)
		if peer.Host == "" {
			return true
		}
		return false
	}()
	if skip {
		return SyncPeerRes{skip: skip}
	}

	log.Printf("net %s peer %s: ensuring client", cnn, pn)
	err := c.ensureClient(ctx, cnn, pn)
	if err != nil {
		return SyncPeerRes{err: fmt.Errorf("ensure client: %w", err)}
	}
	log.Printf("net %s peer %s: pinging", cnn, pn)
	err = c.ping(ctx, cnn, pn)
	if err != nil {
		return SyncPeerRes{err: fmt.Errorf("ping: %w", err)}
	}
	log.Printf("net %s peer %s: pinged", cnn, pn)
	log.Printf("net %s peer %s: authenticating", cnn, pn)
	err = c.auth(ctx, cnn, pn)
	if err != nil {
		return SyncPeerRes{err: fmt.Errorf("auth: %w", err)}
	}
	log.Printf("net %s peer %s: authed", cnn, pn)
	log.Printf("net %s peer %s: exchanging", cnn, pn)
	err = c.xch(ctx, cnn, pn)
	if err != nil {
		return SyncPeerRes{err: fmt.Errorf("xch: %w", err)}
	}
	log.Printf("net %s peer %s: xched", cnn, pn)
	return SyncPeerRes{}
}

func (c *Node) auth(ctx context.Context, cnn string, pn string) (err error) {
	c.serversLock.Lock()
	defer c.serversLock.Unlock()
	log.Print("servers", c.servers)
	cs, ok := c.servers[networkPeerPair{cnn, pn}]
	if !ok {
		return errors.New("corresponding clientServer not found")
	}
	log.Print("preauth")
	conn, err := cs.cl.Auth(ctx)
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	log.Print("postinitauth")
	cn := c.cc.Networks[cnn]
	state := authState{
		coordPrivKey: c.coordPrivKey,
		conn:         conn,
		cc:           c.cc,
		cn:           cn,
		you:          cn.Peers[pn],
	}
	log.Print("preauthboth")
	err = state.verifyChall(cnn, pn)
	if err != nil {
		return fmt.Errorf("verify chall: %w", err)
	}
	err = state.solveChall()
	if err != nil {
		return fmt.Errorf("solve chall: %w", err)
	}

	log.Printf("net %s peer %s: waiting for token", cnn, pn)
	sq, err := conn.Recv()
	if err != nil {
		return err
	}
	token := sq.Sq.(*api.AuthSQ_Token).Token.Token
	cs.token = string(token)
	return nil
}

func (c *Node) xch(ctx context.Context, cnn string, pn string) (err error) {
	cn := c.cc.Networks[cnn]
	peer := cn.Peers[pn]
	// TODO: dont xch if locked?
	peer.lsaLock.Lock()
	defer peer.lsaLock.Unlock()
	if time.Since(peer.lsa) < 1*time.Second {
		return errors.New("attempted to sync too recently")
	}
	peer.lock.Lock()
	log.Printf("LOCK net %s peer %s", cnn, pn)
	defer peer.lock.Unlock()
	cs := c.servers[networkPeerPair{cnn, pn}]
	pubKey := c.cc.Networks[cnn].myPrivKey.PublicKey()
	psk, err := wgtypes.GenerateKey()
	if err != nil {
		return errors.New("PSK generation failed")
	}
	if cs.token == "" {
		return errors.New("blank token")
	}
	s, err := cs.cl.Xch(ctx, &api.XchQ{
		Token:  []byte(cs.token),
		PubKey: pubKey[:],
		Psk:    psk[:],
	})
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	yourPubKey, err := wgtypes.NewKey(s.PubKey)
	if err != nil {
		return errors.New("invalid public key")
	}
	peer.pubKey = &yourPubKey
	peer.psk = &psk
	log.Println("SET1 PSK", peer, "same", psk)
	peer.latestSync = time.Now()
	peer.accessible = true
	return nil
}

func (c *Node) ping(ctx context.Context, cnn string, pn string) (err error) {
	cs := c.servers[networkPeerPair{cnn, pn}]
	_, err = cs.cl.Ping(ctx, &api.PingQS{})
	if err != nil {
		return
	}
	return
}
