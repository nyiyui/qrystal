package node

// TODO: check if all AllowedIPs are in IPs

import (
	"fmt"
	"time"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/cs"
	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/util"
	"gopkg.in/yaml.v3"
)

const backoffTimeout = 5 * time.Minute

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
	// Setup
	cl, err := n.newClient(i)
	if err != nil {
		return
	}

	return false, n.pullCS(i, cl)
}

func (n *Node) pullCS(i int, cl *rpc2.Client) (err error) {
	csc := n.cs[i]
	var s api.PullS
	err = cl.Call("pull", &api.PullQ{CentralToken: csc.Token}, &s)
	if err != nil {
		err = fmt.Errorf("pull init: %w", err)
		return
	}
	cc := s.CC

	n.Kiriyama.SetCS(i, "引き")
	ccy, _ := yaml.Marshal(cc)
	util.S.Infof("新たなCCを受信: %s", ccy)

	// NetworksAllowed
	cc2 := map[string]*central.Network{}
	for cnn, cn := range cc.Networks {
		if csc.netAllowed(cnn) {
			cc2[cnn] = cn
		} else {
			util.S.Warnf("net not allowed; ignoring: %s", cnn)
		}
	}
	cc.Networks = cc2

	err = func() error {
		n.ccLock.Lock()
		defer n.ccLock.Unlock()
		for cnn := range cc.Networks {
			n.csNets[cnn] = i
		}
		toRemove := cs.MissingFromFirst(cc.Networks, n.cc.Networks)
		err = n.removeDevices(toRemove)
		if err != nil {
			return fmt.Errorf("rm all devs: %w", err)
		}
		n.applyCC(&cc)
		n.reify()
		return nil
	}()
	if err != nil {
		return
	}
	return
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
		n.cc.Desynced = central.DNetwork
	}
	for cnn2, cn2 := range cc2.Networks {
		cn, ok := n.cc.Networks[cnn2]
		if !ok {
			// new cn
			n.cc.Networks[cnn2] = &central.Network{}
			cn = n.cc.Networks[cnn2]
		}
		cn.Desynced = 0
		if !central.Same(cn.IPs, cn2.IPs) || !central.Same2(cn.Peers, cn2.Peers) {
			cn.Desynced |= central.DIPs
		}
		cn.Name = cnn2
		cn.IPs = cn2.IPs
		forwardingPeers := map[string]struct{}{}
		if cn.Peers == nil {
			cn.Peers = map[string]*central.Peer{}
		}
		for pn2, peer2 := range cn2.Peers {
			peer, ok := cn.Peers[pn2]
			if !ok {
				// new peer
				cn.Peers[pn2] = peer2
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
			peer.CanSee = peer2.CanSee
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
		n.cc.Desynced = central.DNetwork
	}
}
