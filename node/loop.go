package node

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"time"

	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/node/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func (n *Node) setupCS() (api.CentralSourceClient, error) {
	conn, err := grpc.Dial(n.csHost, grpc.WithTimeout(5*time.Second), grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	if err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}
	cl := api.NewCentralSourceClient(conn)
	_, err = cl.Ping(context.Background(), &api.PingQS{})
	if err != nil {
		return nil, fmt.Errorf("ping %s: %w", n.csHost, err)
	}
	return cl, nil
}

func (n *Node) ListenCS() error {
	cl, err := n.setupCS()
	if err != nil {
		return err
	}
	n.csCl = cl

	if n.azusa.enabled {
		err = n.azusa.setup(n, cl)
		if err != nil {
			return fmt.Errorf("azusa: %w", err)
		}
	}

	conn, err := cl.Pull(context.Background(), &api.PullQ{
		CentralToken: n.csToken,
	})
	if err != nil {
		return fmt.Errorf("pull init: %w", err)
	}

	ctx := conn.Context()
	retryInterval := 1 * time.Second
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("disconnected; retry in %s", retryInterval)
		default:
			s, err := conn.Recv()
			if err != nil {
				return fmt.Errorf("pull recv: %w", err)
			}
			log.Printf("preconv: %s", s.Cc)
			cc, err := newCCFromAPI(s.Cc)
			if err != nil {
				return fmt.Errorf("conv: %w", err)
			}
			cc.DialOpts = []grpc.DialOption{
				grpc.WithTransportCredentials(n.csCreds),
			}
			log.Printf("新たなCCを受信: %#v", cc)
			for cnn, cn := range cc.Networks {
				log.Printf("net %s: %#v", cnn, cn)
			}

			for cnn, cn := range cc.Networks {
				me := cn.Peers[cn.Me]
				log.Printf("my peer not found %s", cn.Me)
				if !bytes.Equal(me.PublicKey, []byte(n.coordPrivKey.Public().(ed25519.PublicKey))) {
					return fmt.Errorf("net %s: key pair mismatch", cnn)
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
			log.Printf("新たなCCで同期します。")
			res, err := n.Sync(context.Background())
			if err != nil {
				return fmt.Errorf("sync: %w", err)
			}
			// TODO: check res
			// TODO: fallback to previous if all fails? perhaps as an option in PullS?
			log.Printf("新たなCCで同期：\n%s", res)
			if err != nil {
				return err
			}
		}
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
