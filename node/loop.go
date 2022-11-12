package node

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

const backoffTimeout = 5 * time.Minute

func (n *Node) setupCS(i int, csc CSConfig) (api.CentralSourceClient, error) {
	conn, err := grpc.Dial(csc.Host, grpc.WithTimeout(5*time.Second), grpc.WithTransportCredentials(csc.Creds))
	if err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}
	cl := api.NewCentralSourceClient(conn)
	_, err = cl.Ping(context.Background(), &api.PingQS{})
	if err != nil {
		return nil, fmt.Errorf("ping %s: %w", csc.Host, err)
	}
	n.Kiriyama.SetCS(i, "ピンOK")
	return cl, nil
}

type listenError struct {
	i   int
	err error
}

func (n *Node) ListenCS() {
	errCh := make(chan listenError, len(n.cs))
	for i := range n.cs {
		go func(i int) {
			errCh <- listenError{i: i, err: n.listenCS(i)}
		}(i)
	}
	select {
	case err := <-errCh:
		csc := n.cs[err.i]
		util.S.Errorf("cs %d (%s at %s) error: %s", err.i, csc.Comment, csc.Host, err.err)
	}
}

func (n *Node) listenCS(i int) error {
	backoff := 1 * time.Second
	n.Kiriyama.SetCS(i, "初期")
	for {
		resetBackoff, err := n.listenCSOnce(i)
		if err == nil {
			continue
		}
		if resetBackoff {
			backoff = 1 * time.Second
		}
		util.S.Errorf("listen: %s; retry in %s", err, backoff)
		util.S.Errorw("listen: error",
			"err", err,
			"backoff", backoff,
		)
		n.Kiriyama.SetCS(i, fmt.Sprintf("%sで再試行", backoff))
		time.Sleep(backoff)
		if backoff <= backoffTimeout {
			backoff *= 2
		}
		if resetBackoff {
			backoff = 1 * time.Second
		}
	}
}

func (n *Node) listenCSOnce(i int) (resetBackoff bool, err error) {
	csc := n.cs[i]

	// Setup
	cl, err := n.setupCS(i, csc)
	if err != nil {
		return
	}
	n.csCls[i] = cl

	// Azusa
	if n.azusa.enabled && i == 0 {
		err = n.azusa.setup(n, csc, cl)
		if err != nil {
			err = fmt.Errorf("azusa: %w", err)
			return
		}
	} else if csc.Azusa != nil {
		nakano := newAzusa(*csc.Azusa)
		err = nakano.setup(n, csc, cl)
		if err != nil {
			err = fmt.Errorf("azusa nakano: %w", err)
			return
		}
	}

	conn, err := cl.Pull(context.Background(), &api.PullQ{
		CentralToken: csc.Token,
	})
	if err != nil {
		err = fmt.Errorf("pull init: %w", err)
		return
	}

	ctx := conn.Context()
	for {
		select {
		case <-ctx.Done():
			err = errors.New("disconnected")
			return
		default:
			var s *api.PullS
			s, err = conn.Recv()
			if err != nil {
				err = fmt.Errorf("pull recv: %w", err)
				return
			}
			n.Kiriyama.SetCS(i, "引き")
			var cc *CentralConfig
			cc, err = newCCFromAPI(s.Cc)
			if err != nil {
				err = fmt.Errorf("conv: %w", err)
				return
			}
			ccy, _ := yaml.Marshal(cc)
			util.S.Infof("新たなCCを受信: %s", ccy)

			// NetworksAllowed
			cc2 := map[string]*CentralNetwork{}
			for cnn, cn := range cc.Networks {
				if csc.netAllowed(cnn) {
					cc2[cnn] = cn
				} else {
					util.S.Warnf("net not allowed; ignoring: %s", cnn)
				}
			}
			cc.Networks = cc2

			for cnn, cn := range cc.Networks {
				me := cn.Peers[cn.Me]
				if !bytes.Equal(me.PublicKey, []byte(n.coordPrivKey.Public().(ed25519.PublicKey))) {
					err = fmt.Errorf("net %s me (%s): key pair mismatch", cn.Me, cnn)
					return
				}
			}

			err = func() error {
				n.ccLock.Lock()
				defer n.ccLock.Unlock()
				for cnn := range cc.Networks {
					n.csNets[cnn] = i
				}
				err = n.removeAllDevices()
				if err != nil {
					return fmt.Errorf("rm all devs: %w", err)
				}
				n.applyCC(cc)
				return nil
			}()
			if err != nil {
				return
			}
			if s.ForwardingOnly {
				n.Kiriyama.SetCS(i, "同期中（フォワード）")
				util.S.Infof("===フォワードだけなので同期しません。")
				var res *SyncRes
				res, err = n.syncBackoff(context.Background(), false)
				if err != nil {
					err = fmt.Errorf("sync: %w", err)
					return
				}
				// TODO: check res
				// TODO: fallback to previous if all fails? perhaps as an option in PullS?
				util.S.Infof("===フォワードだけ：\n%s", res)
				n.Kiriyama.SetCS(i, "同期OK（フォワード）")
			} else {
				n.Kiriyama.SetCS(i, "同期中（新規）")
				util.S.Infof("===新たなCCで同期します。")
				var res *SyncRes
				res, err = n.syncBackoff(context.Background(), true)
				if err != nil {
					err = fmt.Errorf("sync: %w", err)
					return
				}
				// TODO: check res
				// TODO: fallback to previous if all fails? perhaps as an option in PullS?
				util.S.Infof("===新たなCCで同期：\n%s", res)
				n.Kiriyama.SetCS(i, "同期OK（新規）")
			}
			resetBackoff = true
		}
	}
}

func (n *Node) syncBackoff(ctx context.Context, xch bool) (*SyncRes, error) {
	backoff := 1 * time.Second
	tryNum := 1
	for {
		// TODO: don't increase backoff if succees for a while
		util.S.Infof("sync starting: try num %d", tryNum)
		res, err := n.Sync(ctx, xch)
		if err != nil || res.allOK() {
			return res, nil
		}
		if err != nil {
			util.S.Errorf("sync: %s; retry in %s", err, backoff)
		} else {
			util.S.Errorf("sync res: %s; retry in %s", res, backoff)
		}
		util.S.Errorw("sync: error",
			"err", err,
			"res", res,
			"backoff", backoff,
		)
		time.Sleep(backoff)
		if backoff <= backoffTimeout {
			backoff *= 2
		}
		tryNum++
	}
}

func (c *Node) removeAllDevices() error {
	for nn := range c.cc.Networks {
		err := c.mio.RemoveDevice(mio.RemoveDeviceQ{
			Name: nn,
		})
		if err != nil {
			return fmt.Errorf("mio RemoveDevice %s: %w", nn, err)
		}
	}
	return nil
}

// applyCC applies cc2 to n.
// ccLock is locked by the caller
func (n *Node) applyCC(cc2 *CentralConfig) {
	// NOTE: shouldn't have to lock any more since ccLock is supposed to override all inner locks
	if n.cc.Networks == nil {
		n.cc.Networks = map[string]*CentralNetwork{}
	}
	for cnn2, cn2 := range cc2.Networks {
		cn, ok := n.cc.Networks[cnn2]
		if !ok {
			// new cn
			n.cc.Networks[cnn2] = &CentralNetwork{}
			cn = n.cc.Networks[cnn2]
		}
		cn.name = cnn2
		cn.IPs = cn2.IPs
		forwardingPeers := map[string]struct{}{}
		if cn.Peers == nil {
			cn.Peers = map[string]*CentralPeer{}
		}
		for pn2, peer2 := range cn2.Peers {
			peer, ok := cn.Peers[pn2]
			if !ok {
				// new peer
				cn.Peers[pn2] = peer2
				continue
			}
			peer.name = pn2
			peer.Host = peer2.Host
			peer.AllowedIPs = peer2.AllowedIPs
			util.S.Debugf("LOOP net %s peer %s ForwardingPeers1 %s", cnn2, pn2, peer.ForwardingPeers)
			util.S.Debugf("LOOP net %s peer %s ForwardingPeers2 %s", cnn2, pn2, peer2.ForwardingPeers)
			peer.ForwardingPeers = []string{}
			for _, forwardingPeer := range peer2.ForwardingPeers {
				if forwardingPeer == cn.Me {
					continue
				}
				_, ok := forwardingPeers[forwardingPeer]
				if !ok {
					peer.ForwardingPeers = append(peer.ForwardingPeers, forwardingPeer)
					forwardingPeers[forwardingPeer] = struct{}{}
				}
			}
			peer.PublicKey = peer2.PublicKey
			peer.CanForward = peer2.CanForward
		}
		for pn := range cn.Peers {
			_, ok := cn2.Peers[pn]
			if !ok {
				// removed
				delete(cn.Peers, pn)
			}
		}
		cn.Me = cn2.Me
		cn.Keepalive = cn2.Keepalive
		cn.ListenPort = cn2.ListenPort
	}
	for cnn := range n.cc.Networks {
		_, ok := cc2.Networks[cnn]
		if !ok {
			// removed
			delete(n.cc.Networks, cnn)
		}
	}
}
