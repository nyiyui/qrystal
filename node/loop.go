package node

// TODO: check if all AllowedIPs are in IPs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/util"
)

type listenError struct {
	err error
}

func (n *Node) ListenCS() {
	n.reupdateSRV = make(chan string)
	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, syscall.SIGUSR1)
		for range sig {
			n.reupdateSRV <- "received signal"
		}
	}()
	go func() {
		err := n.handleSRV()
		util.S.Errorf("srv error: %s", err)
		panic(fmt.Sprintf("srv error: %s", err))
	}()
	util.S.Debug("listening…")
	util.Notify("STATUS=connecting to CS")
	err := n.listenCS()
	util.S.Errorf("cs listen error: %s", err)
}

func (n *Node) handleSRV() error {
	return util.Backoff(n.handleSRVOnce, func(backoff time.Duration, err error) error {
		util.S.Errorf("srv: %s; retry in %s", err, backoff)
		util.S.Errorw("srv: error",
			"err", err,
			"backoff", backoff,
		)
		return nil
	})
}

func (n *Node) handleSRVOnce() (resetBackoff bool, err error) {
	util.S.Debug("handleSRVOnce: newClient…")
	ctx, cancel := context.WithTimeout(context.Background(), util.OnceTimeout)
	defer cancel()
	cl, _, err := n.newClient(ctx)
	if err != nil {
		return false, fmt.Errorf("newClient: %w", err)
	}

	util.S.Debug("handleSRVOnce: initial")
	err = n.loadSRVList(cl)
	if err != nil {
		err = fmt.Errorf("srv (initial): %w", err)
		return
	}

	for range n.reupdateSRV {
		util.S.Debug("handleSRVOnce: reupdate")
		err = n.loadSRVList(cl)
		if err != nil {
			err = fmt.Errorf("srv (signal): %w", err)
			return
		}
	}
	return true, nil
}

func (n *Node) listenCS() error {
	return util.Backoff(n.listenCSOnce, func(backoff time.Duration, err error) error {
		util.Notify(fmt.Sprintf("STATUS=connecting to CS: %s (retrying in %s)", err, backoff))
		util.S.Errorf("listen: %s; retry in %s", err, backoff)
		util.S.Errorw("listen: error",
			"err", err,
			"backoff", backoff,
		)
		return nil
	})
}

func (n *Node) listenCSOnce() (resetBackoff bool, err error) {
	util.S.Debug("listenCSOnce: newClient…")
	ctx, cancel := context.WithTimeout(context.Background(), util.OnceTimeout)
	defer cancel()
	cl, _, err := n.newClient(ctx)
	if err != nil {
		return false, fmt.Errorf("newClient: %w", err)
	}

	err = cl.CallWithContext(ctx, "ping", new(bool), new(bool))
	if err != nil {
		return false, fmt.Errorf("ping: %w", err)
	}

	util.S.Debug("listenCSOnce: pullCS…")
	err = n.pullCS(ctx, cl)
	if err != nil {
		return false, fmt.Errorf("pullCS: %w", err)
	}
	return true, nil
}

func (n *Node) pullCS(ctx context.Context, cl *rpc2.Client) (err error) {
	if len(n.cs.Azusa) != 0 {
		err = n.azusa(ctx, n.cs.Azusa, cl)
		if err != nil {
			err = fmt.Errorf("azusa: %w", err)
			return
		}
		n.reupdateSRV <- "azusa"
	}
	for {
		func() {
			ctx2, cancel := context.WithCancelCause(context.Background())
			defer cancel(errors.New("cleanup"))
			secret, notify := n.addKeepaliveEntry()
			t := time.NewTimer(util.OnceTimeout)
			go func() {
				select {
				case <-t.C:
					cancel(errors.New("timeout"))
					n.removeKeepaliveEntry(secret)
				case <-notify:
					if !t.Stop() {
						<-t.C
					}
				}
			}()
			var s api.SyncS
			err = cl.CallWithContext(ctx2, "sync", &api.SyncQ{CentralToken: n.cs.Token}, &s)
			if err != nil {
				err = fmt.Errorf("sync: %w", err)
				return
			}
		}()
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
