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
				n.replaceCC(cc)
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
func (n *Node) replaceCC(cc2 *CentralConfig) {
	n.cc = *cc2
}
