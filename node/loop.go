package node

// TODO: check if all AllowedIPs are in IPs

import (
	"fmt"
	"time"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
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
	return util.Backoff(func() (resetBackoff bool, err error) {
		return n.listenCSOnce(i)
	}, func(backoff time.Duration, err error) error {
		util.S.Errorf("listen: %s; retry in %s", err, backoff)
		util.S.Errorw("listen: error",
			"err", err,
			"backoff", backoff,
		)
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
		return false, fmt.Errorf("pullCS: %w", err)
	}
	return true, nil
}

func (n *Node) pullCS(i int, cl *rpc2.Client) (err error) {
	csc := n.cs[i]
	if len(csc.Azusa) != 0 {
		err = n.azusa(i, csc.Azusa, cl)
		if err != nil {
			err = fmt.Errorf("azusa: %w", err)
			return
		}
	}
	for {
		var s api.SyncS
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
