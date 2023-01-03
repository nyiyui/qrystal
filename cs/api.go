package cs

import (
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/util"
)

func (c *CentralSource) newHandler() {
	h := rpc2.NewServer()
	h.Handle("sync", c.sync)
	h.Handle("azusa", c.azusa)
	c.handler = h
}

func (c *CentralSource) sync(cl *rpc2.Client, q *api.SyncQ, s *api.SyncS) error {
	ti, ok, err := c.Tokens.getToken(q.CentralToken)
	if err != nil {
		return err
	}
	if !ok {
		return newTokenAuthError(q.CentralToken)
	}
	if !ti.CanPull {
		return fmt.Errorf("token %s cannot pull", ti.Name)
	}
	ti.StartUse()
	err = c.Tokens.UpdateToken(ti)
	if err != nil {
		return err
	}
	defer func() {
		ti.StopUse()
		err = c.Tokens.UpdateToken(ti)
		if err != nil {
			util.S.Errorf("UpdateToken %s: %s", ti.sum, err)
		}
	}()
	util.S.Infof("%sから新たな認証済プル", ti.Name)

	newCC, err := c.copyCC(ti.Networks)
	if err != nil {
		util.S.Errorf("copyCC: %s", err)
		return errors.New("copyCC failed")
	}

	var s2 api.PushS
	err = cl.Call("push", &api.PushQ{CC: *newCC}, &s2)
	if err != nil {
		util.S.Errorf("push: %s", err)
		return fmt.Errorf("push failed: %w", err)
	}
	util.S.Infof("push result %s", s2)
	changed := map[string][]string{}
	sendNotify := false
	func() {
		c.ccLock.Lock()
		defer c.ccLock.Unlock()
		for cnn, key := range s2.PubKeys {
			cn := c.cc.Networks[cnn]
			pn := ti.Networks[cnn]
			peer := cn.Peers[pn]
			util.S.Debug("l70", cnn, key, cn, pn, peer)
			if peer.Internal.PubKey == nil || *peer.Internal.PubKey != key {
				peer.Internal.PubKey = &key
				changed[cnn] = []string{pn}
				sendNotify = true
				util.S.Infof("net %s peer %s: new/diff PublicKey %s", cnn, ti.Networks[cnn], key)
			}
		}
	}()
	if sendNotify {
		c.notify(change{Reason: fmt.Sprintf("token %s", ti.Name), Changed: changed})
	}

	chI, ch := c.newNotifyCh()
	defer c.removeNotifyCh(chI)
	for chg := range ch {
		affectsYou := func(chg change) bool {
			c.ccLock.RLock()
			defer c.ccLock.RUnlock()
			for cnn, _ := range ti.Networks {
				peers := chg.Changed[cnn]
				cn, ok := newCC.Networks[cnn]
				if !ok {
					continue
				}
				for _, pn2 := range peers {
					if _, ok := cn.Peers[pn2]; ok {
						return true
					}
				}
			}
			return false
		}(chg)
		if affectsYou {
			util.S.Infof("token %s resync due to change %s", ti.Name, chg)
			break
		}
	}
	// TODO: check if changes include peers that this Node can see

	// Nodes will retry pulling when sync is done (if err == nil then with a
	// zeroed backoff), so return when we want the Nodes to resync.

	return nil
}

type change struct {
	Reason  string
	Changed map[string][]string
}

func (c *CentralSource) newNotifyCh() (i int, ch chan change) {
	ch = make(chan change)
	c.notifyChsLock.Lock()
	defer c.notifyChsLock.Unlock()
	i = len(c.notifyChs)
	c.notifyChs = append(c.notifyChs, ch)
	return
}

func (c *CentralSource) removeNotifyCh(i int) {
	c.notifyChsLock.Lock()
	defer c.notifyChsLock.Unlock()
	// TODO: memory leak (chan leak) due to chans not being cleaned up
	c.notifyChs[i] = nil
}

func (c *CentralSource) notify(chg change) {
	c.notifyChsLock.Lock()
	defer c.notifyChsLock.Unlock()
	util.S.Infof("notify: %s", chg)
	for _, ch := range c.notifyChs {
		t := time.NewTimer(1 * time.Second)
		select {
		case ch <- chg:
		case <-t.C:
		}
	}
	c.backportSilent()
}
