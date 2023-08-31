package node

// TODO: check if all AllowedIPs are in IPs

import (
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
	go func() {
		err := n.handleSRV()
		util.S.Errorf("srv error: %s", err)
		panic(fmt.Sprintf("srv error: %s", err))
	}()
	util.S.Debug("listening…")
	err := n.listenCS()
	util.S.Errorf("cs listen error: %s", err)
}

func (n *Node) handleSRV() error {
	reloadCh := make(chan os.Signal)
	signal.Notify(reloadCh, syscall.SIGUSR1)
	return util.Backoff(func() (resetBackoff bool, err error) { return n.handleSRVOnce(reloadCh) }, func(backoff time.Duration, err error) error {
		util.S.Errorf("srv: %s; retry in %s", err, backoff)
		util.S.Errorw("srv: error",
			"err", err,
			"backoff", backoff,
		)
		return nil
	})
}

func (n *Node) handleSRVOnce(reloadCh <-chan os.Signal) (resetBackoff bool, err error) {
	util.S.Debug("newClient…")
	cl, _, err := n.newClient()
	if err != nil {
		return false, fmt.Errorf("newClient: %w", err)
	}

	for range reloadCh {
		err = n.loadSRVList(cl)
		if err != nil {
			err = fmt.Errorf("srv: %w", err)
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
	util.S.Debug("newClient…")
	cl, _, err := n.newClient()
	if err != nil {
		return false, fmt.Errorf("newClient: %w", err)
	}

	util.S.Debug("pullCS…")
	err = n.pullCS(cl)
	if err != nil {
		return false, fmt.Errorf("pullCS: %w", err)
	}
	return true, nil
}

func (n *Node) pullCS(cl *rpc2.Client) (err error) {
	if len(n.cs.Azusa) != 0 {
		err = n.azusa(n.cs.Azusa, cl)
		if err != nil {
			err = fmt.Errorf("azusa: %w", err)
			return
		}
	}
	for {
		var s api.SyncS
		err = cl.Call("sync", &api.SyncQ{CentralToken: n.cs.Token}, &s)
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
