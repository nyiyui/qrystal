package cs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/util"
	"golang.org/x/exp/slices"
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

	newCC := c.copyCC(ti.Networks) // also does RLock/RUnlock

	var s2 api.PushS
	err = cl.Call("push", &api.PushQ{CC: *newCC}, &s2)
	if err != nil {
		util.S.Errorf("push %s nets %s: %s", ti.Name, ti.Networks, err)
		return fmt.Errorf("push failed: %w", err)
	}
	util.S.Debugf("push %s result %s", ti.Name, s2)
	{
		changed := map[string][]string{}
		func() {
			sendNotify := false
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
			if sendNotify {
				c.notify(change{Reason: fmt.Sprintf("token %s", ti.Name), Changed: changed})
			}
		}()
	}

	data, _ := json.Marshal(c.cc)
	util.S.Infof("token %s listening for notifs... %s", ti.Name, data)
	chI, ch := c.newNotifyCh(fmt.Sprintf("token %s", ti.Name))
	defer c.removeNotifyCh(chI)

	// make sure we catch somehow-not-caught desync as well
	desynced := c.resetSynced(ti)
	if desynced {
		util.S.Infof("token %s resync after pull", ti.Name)
		return nil // see below
	}

	for chg := range ch {
		desynced := c.resetSynced(ti)
		if desynced {
			util.S.Infof("token %s resync due to change %s", ti.Name, chg)
			break // see below
		} else {
			util.S.Infof("token %s up-to-date including %s; change ignored", chg)
		}
	}

	// Nodes will retry pulling when sync is done (if err == nil then with a
	// zeroed backoff), so return when we want the Nodes to resync.
	return nil
}

func (c *CentralSource) resetSynced(ti TokenInfo) (wasDesynced bool) {
	myCC := c.copyCC(ti.Networks)
	desynced := false
	func() {
		c.ccLock.RLock()
		defer c.ccLock.RUnlock()
		for cnn, myCN := range myCC.Networks {
			me := myCN.Me
			for pn, peer := range myCN.Peers {
				util.S.Infof("resetSynced token %s: looking in net %s peer %s: %#v", ti.Name, cnn, pn, peer.SyncedPeers)
				if !slices.Contains(peer.SyncedPeers, me) {
					desynced = true
					peer.SyncedPeers = append(peer.SyncedPeers, me)
				}
			}
		}
	}()
	return desynced
}

type change struct {
	// Reason is only for debugging, and the reason for the change is stored here.
	// For example: "token token_name changed peer_name"
	Reason string
	// Changed is a map with key = CN name and value = list of peers in the CN that changed.
	Changed map[string][]string
}
type notifyCh struct {
	Ch      chan change
	Comment string
}

func (c *CentralSource) newNotifyCh(comment string) (i int, ch chan change) {
	ch = make(chan change)
	c.notifyChsLock.Lock()
	defer c.notifyChsLock.Unlock()
	i = len(c.notifyChs)
	c.notifyChs = append(c.notifyChs, notifyCh{
		ch, comment,
	})
	return
}

func (c *CentralSource) removeNotifyCh(i int) {
	c.notifyChsLock.Lock()
	defer c.notifyChsLock.Unlock()
	close(c.notifyChs[i].Ch)
	// TODO: memory leak (chan leak) due to chans not being cleaned up
	c.notifyChs[i] = notifyCh{}
}

func (c *CentralSource) notify(chg change) {
	// NOTE: c.ccLock must be taken!
	c.notifyChsLock.Lock()
	defer c.notifyChsLock.Unlock()
	for cnn, pns := range chg.Changed {
		cn, ok := c.cc.Networks[cnn]
		if !ok {
			panic("CentralSource.notify: change contains nonexistent CN - was CC changed after change{} was made and CentralSource.notify was called?")
		}
		for _, pn := range pns {
			util.S.Infof("notify: resetting SyncedPeers for net %s peer %s", cnn, pn)
			cn.Peers[pn].SyncedPeers = nil
		}
	}
	c.backportSilent()
	util.S.Infof("sending notify: %s", chg)
	for _, nch := range c.notifyChs {
		if nch == (notifyCh{}) {
			// skip deleted ones
			continue
		}
		t := time.NewTimer(1 * time.Second)
		select {
		case nch.Ch <- chg:
			util.S.Warnf("notify sent on %s: %s", nch.Comment, chg)
		case <-t.C:
			util.S.Warnf("notify timeout on %s: %s", nch.Comment, chg)
		}
	}
}
