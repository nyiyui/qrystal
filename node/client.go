package node

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/node/verify"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
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

func (c *Node) Sync(ctx context.Context, xch bool, changedCNs []string) (*SyncRes, error) {
	res := SyncRes{
		netStatus: map[string]SyncNetRes{},
	}
	c.ccLock.RLock()
	defer c.ccLock.RUnlock()
	var cnns []string
	if changedCNs == nil {
		cnns = make([]string, 0, len(c.cc.Networks))
		for cnn := range c.cc.Networks {
			cnns = append(cnns, cnn)
		}
	} else {
		cnns = changedCNs
	}
	var wg sync.WaitGroup
	for _, cnn := range cnns {
		wg.Add(1)
		go func(cnn string) {
			defer wg.Done()
			netRes, err := c.syncNetwork(ctx, cnn, xch)
			if netRes == nil {
				netRes = &SyncNetRes{}
			}
			netRes.err = err
			res.netStatus[cnn] = *netRes
		}(cnn)
	}
	wg.Wait()
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
					c.Kiriyama.SetPeer(cn.Name, pn, "接続中")
					err := c.ensureClient(ctx, cnn, pn)
					if err != nil {
						res.peerStatus[pn] = SyncPeerRes{
							err: err,
						}
						return
					}
					c.Kiriyama.SetPeer(cn.Name, pn, "交換中")
					ps := c.xchPeer(ctx, cnn, pn)
					c.Kiriyama.SetPeer(cn.Name, pn, "交換OK")
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
		util.S.Debugf("net %s peers %s advertising forwarding capability", cn.Name, pns)
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
		peer.Lock.RLock()
		defer peer.Lock.RUnlock()
		if peer.Host == "" {
			return true
		}
		return false
	}()
	if skip {
		return SyncPeerRes{skip: skip}
	}

	util.S.Debugf("net %s peer %s: ensuring client", cnn, pn)
	err := c.ensureClient(ctx, cnn, pn)
	if err != nil {
		return SyncPeerRes{err: fmt.Errorf("ensure client: %w", err)}
	}
	util.S.Debugf("net %s peer %s: pinging", cnn, pn)
	err = c.ping(ctx, cnn, pn)
	if err != nil {
		return SyncPeerRes{err: fmt.Errorf("ping: %w", err)}
	}
	util.S.Debugf("net %s peer %s: pinged", cnn, pn)
	util.S.Debugf("net %s peer %s: exchanging", cnn, pn)
	err = c.xch(ctx, cnn, pn)
	if err != nil {
		return SyncPeerRes{err: fmt.Errorf("xch: %w", err)}
	}
	util.S.Debugf("net %s peer %s: xched", cnn, pn)
	return SyncPeerRes{}
}

func (c *Node) xch(ctx context.Context, cnn string, pn string) (err error) {
	cn := c.cc.Networks[cnn]
	peer := cn.Peers[pn]
	// TODO: dont xch if locked?
	peer.LSALock.Lock()
	defer peer.LSALock.Unlock()
	if time.Since(peer.LSA) < 1*time.Second {
		return errors.New("attempted to sync too recently")
	}
	peer.Lock.Lock()
	defer peer.Lock.Unlock()
	cs := c.servers[networkPeerPair{cnn, pn}]
	pubKey := c.cc.Networks[cnn].MyPrivKey.PublicKey()
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

	err = verify.VerifyXchS(ed25519.PublicKey(peer.PublicKey), s)
	if err != nil {
		return fmt.Errorf("VerifyXchS: %w", err)
	}

	yourPubKey, err := wgtypes.NewKey(s.PubKey)
	if err != nil {
		return errors.New("invalid public key")
	}
	peer.PubKey = &yourPubKey
	peer.PSK = &psk
	peer.LatestSync = time.Now()
	peer.Accessible = true
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
