package cs

import (
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (c *CentralSource) newHandler() {
	h := rpc2.NewServer()
	h.Handle("ping", c.ping)
	h.Handle("sync", c.sync)
	h.Handle("azusa", c.azusa)
	h.Handle("srvUpdate", c.srvUpdate)
	c.handler = h
}

func (c *CentralSource) ping(cl *rpc2.Client, q *bool, s *bool) error {
	return nil
}

func (c *CentralSource) sync(cl *rpc2.Client, q *api.SyncQ, s *api.SyncS) error {
	ti, ok, err := c.Tokens.getToken(&q.CentralToken)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
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
		return fmt.Errorf("update token: %w", err)
	}
	defer func() {
		ti.StopUse()
		err = c.Tokens.UpdateToken(ti)
		if err != nil {
			util.S.Errorf("UpdateToken %s: %s", ti.key, err)
		}
	}()
	util.S.Infof("%sからプル。ネットワーク：%s", ti.Name, ti.Networks)

	{
		var q, s bool
		err = cl.Call("ping", &q, &s)
		if err != nil {
			util.S.Errorf("ping token %s: %s", ti.Name, err)
			return fmt.Errorf("ping failed: %w", err)
		}
	}

	newCC, err := c.copyCC(ti.Networks)
	if err != nil {
		util.S.Errorf("copyCC: %s", err)
		return errors.New("copyCC failed")
	}

	// NOTE: race condition around here - network can change between after push call (here) and newNotifyCh
	// call newNotifyCh before push call so we don't have race conditon where:
	// - node1 pulls
	// - node1 pulls
	// - node2 push starts and node1 push ends (new PubKeys notified)
	// - node2 push ends
	//   - ※ here, node2 has not called newNotifyCh yet, so doesn't reload
	chI, ch := c.newNotifyCh()
	defer c.removeNotifyCh(chI)

	var s2 api.PushS
	err = cl.Call("push", &api.PushQ{CC: *newCC}, &s2)
	if err != nil {
		util.S.Errorf("push %s nets %s: %s", ti.Name, ti.Networks, err)
		return fmt.Errorf("push failed: %w", err)
	}
	util.S.Debugf("push %s result %s", ti.Name, s2)
	{
		changed := map[string][]string{}
		sendNotify := false
		func() {
			c.ccLock.Lock()
			defer c.ccLock.Unlock()
			for cnn, key := range s2.PubKeys {
				cn := c.cc.Networks[cnn]
				pn := ti.Networks[cnn]
				peer := cn.Peers[pn]
				util.S.Debugf("l70 %s %s key %s net %s peer %s", cnn, pn, key, cn, peer)
				if peer.PubKey == (wgtypes.Key{}) || peer.PubKey != key {
					peer.PubKey = key
					changed[cnn] = []string{pn}
					sendNotify = true
					util.S.Infof("net %s peer %s: new/diff PublicKey %s", cnn, ti.Networks[cnn], key)
				}
				util.S.Debugf("l80 %s %s key %s net %s peer %s", cnn, pn, key, cn, peer)
			}
		}()
		if sendNotify {
			c.notify(change{Reason: fmt.Sprintf("token %s", ti.Name), Changed: changed})
		}
	}

	for chg := range ch {
		newCC2, err := c.copyCC(ti.Networks)
		if err != nil {
			util.S.Errorf("newCC2: %s", err)
			return errors.New("newCC2 failed")
		}
		util.S.Debugf("newCC2: %s", newCC2)
		affectsYou := func(chg change) bool {
			c.ccLock.RLock()
			defer c.ccLock.RUnlock()
			for cnn := range ti.Networks {
				chgPeers := chg.Changed[cnn]
				cn, ok := newCC2.Networks[cnn]
				if !ok {
					continue
				}
				for _, pn2 := range chgPeers {
					if _, ok := cn.Peers[pn2]; ok {
						return true
					}
				}
			}
			return false
		}(chg)
		if affectsYou {
			util.S.Infof("token %s resync due to change %s", ti.Name, chg)
			break // see below
		}
	}

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
