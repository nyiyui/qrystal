package node

// TODO: check if all AllowedIPs are in IPs

import (
	"fmt"
	"time"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/util"
)

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
	err := <-errCh
	csc := n.cs[err.i]
	util.S.Errorf("cs %d (%s at %s) error: %s", err.i, csc.Comment, csc.Host, err.err)
}

func (n *Node) listenCS(i int) error {
	n.Kiriyama.SetCS(i, "初期")
	return util.Backoff(func() (resetBackoff bool, err error) {
		return n.listenCSOnce(i)
	}, func(backoff time.Duration, err error) error {
		util.S.Errorf("listen: %s; retry in %s", err, backoff)
		util.S.Errorw("listen: error",
			"err", err,
			"backoff", backoff,
		)
		n.Kiriyama.SetCS(i, fmt.Sprintf("%sで再試行", backoff))
		return nil
	})
}

func (n *Node) listenCSOnce(i int) (resetBackoff bool, err error) {
	// Setup
	util.S.Debug("newClient…")
	cl, _, err := n.newClient(i)
	if err != nil {
		return
	}

	util.S.Debug("pullCS…")
	err = n.pullCS(i, cl)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (n *Node) pullCS(i int, cl *rpc2.Client) (err error) {
	csc := n.cs[i]
	err = n.azusa(i, csc.Azusa, cl)
	if err != nil {
		err = fmt.Errorf("azusa: %w", err)
		return
	}
	for {
		var s api.SyncS
		n.Kiriyama.SetCS(i, "引き")
		err = cl.Call("sync", &api.SyncQ{I: i, CentralToken: csc.Token}, &s)
		if err != nil {
			err = fmt.Errorf("sync: %w", err)
			return
		}
	}
}

func (c *Node) removeDevices(devices []string) error {
	for _, nn := range devices {
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
func (n *Node) applyCC(cc2 *central.Config) {
	// NOTE: shouldn't have to lock any more since ccLock is supposed to override all inner locks
	if n.cc.Networks == nil {
		n.cc.Networks = map[string]*central.Network{}
		n.cc.Desynced |= central.DNetwork
	}
	for cnn2, cn2 := range cc2.Networks {
		cn, ok := n.cc.Networks[cnn2]
		if !ok {
			// new cn
			n.cc.Networks[cnn2] = &central.Network{}
			cn = n.cc.Networks[cnn2]
			n.cc.Desynced |= central.DNetwork
		}
		cn.Desynced = 0
		cn.Name = cnn2
		cn.IPs = cn2.IPs
		forwardingPeers := map[string]struct{}{}
		if cn.Peers == nil {
			cn.Peers = map[string]*central.Peer{}
			cn.Desynced |= central.DIPs
		}
		if !central.Same(cn.IPs, cn2.IPs) || !central.Same2(cn.Peers, cn2.Peers) {
			cn.Desynced |= central.DIPs
		}
		for pn2, peer2 := range cn2.Peers {
			peer, ok := cn.Peers[pn2]
			if !ok {
				// new peer
				peer2.Name = pn2
				cn.Peers[pn2] = peer2
				cn.Desynced |= central.DIPs
				continue
			}
			peer.Desynced = 0
			if !peer.Same(peer2) {
				peer.Desynced = central.DPeer
			}
			peer.Name = pn2
			peer.Host = peer2.Host
			peer.AllowedIPs = peer2.AllowedIPs
			if !central.Same(peer.AllowedIPs, peer2.AllowedIPs) {
				peer.Desynced |= central.DIPs
			}
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
			peer.CanSee = peer2.CanSee
			if peer.Internal == nil {
				peer.Internal = new(central.PeerInternal)
			}
			if peer.Internal.PubKey != peer2.Internal.PubKey {
				peer.Desynced |= central.DKeys
			}
			peer.Internal.PubKey = peer2.Internal.PubKey
			cn.Peers[pn2] = peer
		}
		for pn := range cn.Peers {
			cn.Desynced |= central.DIPs
			_, ok := cn2.Peers[pn]
			if !ok {
				// removed
				delete(cn.Peers, pn)
			}
		}
		cn.Me = cn2.Me
		cn.Keepalive = cn2.Keepalive
		cn.ListenPort = cn2.ListenPort
		n.cc.Networks[cnn2] = cn
	}
	for cnn := range n.cc.Networks {
		_, ok := cc2.Networks[cnn]
		if !ok {
			// removed
			delete(n.cc.Networks, cnn)
		}
		n.cc.Desynced |= central.DNetwork
	}
}
