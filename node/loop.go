package node

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"google.golang.org/grpc"
)

func (n *Node) setupCS(csc CSConfig) (api.CentralSourceClient, error) {
	conn, err := grpc.Dial(csc.Host, grpc.WithTimeout(5*time.Second), grpc.WithTransportCredentials(csc.Creds))
	if err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}
	cl := api.NewCentralSourceClient(conn)
	_, err = cl.Ping(context.Background(), &api.PingQS{})
	if err != nil {
		return nil, fmt.Errorf("ping %s: %w", csc.Host, err)
	}
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
		log.Printf("cs %d (%s at %s) error: %s", err.i, csc.Comment, csc.Host, err.err)
	}
}

func (n *Node) listenCS(i int) error {
	backoff := 1 * time.Second
RetryLoop:
	for {
		// TODO: don't increase backoff if succees for a while
		err := n.listenCSOnce(i)
		if err == nil {
			continue
		}
		util.S.Errorf("listen: %s; retry in %s", err, backoff)
		util.S.Errorw("listen: error",
			"err", err,
			"backoff", backoff,
		)
		time.Sleep(backoff)
		backoff *= 2
		if backoff > 65536*time.Second {
			break RetryLoop
		}
	}
	return errors.New("backoff exceeded")
}

func (n *Node) listenCSOnce(i int) error {
	csc := n.cs[i]

	// Setup
	cl, err := n.setupCS(csc)
	if err != nil {
		return err
	}
	n.csCls[i] = cl

	// Azusa
	if n.azusa.enabled {
		err = n.azusa.setup(n, csc, cl)
		if err != nil {
			return fmt.Errorf("azusa: %w", err)
		}
	}

	conn, err := cl.Pull(context.Background(), &api.PullQ{
		CentralToken: csc.Token,
	})
	if err != nil {
		return fmt.Errorf("pull init: %w", err)
	}

	ctx := conn.Context()
	for {
		select {
		case <-ctx.Done():
			return errors.New("disconnected")
		default:
			s, err := conn.Recv()
			if err != nil {
				return fmt.Errorf("pull recv: %w", err)
			}
			util.S.Infof("次を受信: %s", s)
			cc, err := newCCFromAPI(s.Cc)
			if err != nil {
				return fmt.Errorf("conv: %w", err)
			}
			util.S.Infof("===新たなCCを受信: %#v", cc)
			for cnn, cn := range cc.Networks {
				log.Printf("net %s: %#v", cnn, cn)
			}

			// NetworksAllowed
			cc2 := map[string]*CentralNetwork{}
			for cnn, cn := range cc.Networks {
				if csc.netAllowed(cnn) {
					cc2[cnn] = cn
				} else {
					log.Printf("net not allowed; ignoring: %s", cnn)
				}
			}
			cc.Networks = cc2

			for cnn, cn := range cc.Networks {
				me := cn.Peers[cn.Me]
				if !bytes.Equal(me.PublicKey, []byte(n.coordPrivKey.Public().(ed25519.PublicKey))) {
					return fmt.Errorf("net %s me (%s): key pair mismatch", cn.Me, cnn)
				}
			}

			err = func() error {
				n.ccLock.Lock()
				defer n.ccLock.Unlock()
				err = n.removeAllDevices()
				if err != nil {
					return fmt.Errorf("rm all devs: %w", err)
				}
				n.applyCC(cc)
				return nil
			}()
			if err != nil {
				return err
			}
			if s.ForwardingOnly {
				log.Printf("===フォワードだけなので同期しません。")
				res, err := n.syncBackoff(context.Background(), false)
				if err != nil {
					return fmt.Errorf("sync: %w", err)
				}
				// TODO: check res
				// TODO: fallback to previous if all fails? perhaps as an option in PullS?
				log.Printf("===フォワードだけ：\n%s", res)
			} else {
				log.Printf("===新たなCCで同期します。")
				res, err := n.syncBackoff(context.Background(), true)
				if err != nil {
					return fmt.Errorf("sync: %w", err)
				}
				// TODO: check res
				// TODO: fallback to previous if all fails? perhaps as an option in PullS?
				log.Printf("===新たなCCで同期：\n%s", res)
			}
		}
	}
}

func (n *Node) syncBackoff(ctx context.Context, xch bool) (*SyncRes, error) {
	backoff := 1 * time.Second
RetryLoop:
	for {
		// TODO: don't increase backoff if succees for a while
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
		backoff *= 2
		if backoff > 65536*time.Second {
			break RetryLoop
		}
	}
	return nil, errors.New("backoff exceeded")
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
			n.cc.Networks[cnn2] = cn2
			continue
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
			peer.ForwardingPeers = []string{}
			for _, forwardingPeer := range peer2.ForwardingPeers {
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
