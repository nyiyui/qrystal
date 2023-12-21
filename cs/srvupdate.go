package cs

import (
	"fmt"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
)

func (c *CentralSource) srvUpdate(cl *rpc2.Client, q *api.SRVUpdateQ, s *api.SRVUpdateS) error {
	ti, ok, err := c.Tokens.getToken(&q.CentralToken)
	if err != nil {
		return err
	}
	if !ok {
		return newTokenAuthError(q.CentralToken)
	}
	for _, srv := range q.SRVs {
		cn, ok := c.cc.Networks[srv.NetworkName]
		if !ok {
			return fmt.Errorf("net %s no exist :(", srv.NetworkName)
		}
		_, ok = cn.Peers[srv.PeerName]
		if !ok {
			return fmt.Errorf("net %s peer %s no exist :(", srv.NetworkName, srv.PeerName)
		}
		err = checkSrv(ti, srv.NetworkName)
		if err != nil {
			return err
		}
	}
	util.S.Infof("srvUpdate from token %s to push %d:\n%#v", ti.Name, len(q.SRVs), q.SRVs)
	ti.StartUse()
	err = c.Tokens.UpdateToken(ti)
	if err != nil {
		return err
	}
	defer func() {
		ti.StopUse()
		err = c.Tokens.UpdateToken(ti)
		if err != nil {
			util.S.Errorf("UpdateToken %s: %s", ti.key, err)
		}
	}()
	c.ccLock.Lock()
	defer c.ccLock.Unlock()
	changed := map[string][]string{}
	for srvI, srv := range q.SRVs {
		cn, ok := c.cc.Networks[srv.NetworkName]
		if !ok {
			return fmt.Errorf("net %s no exist :(", srv.NetworkName)
		}
		peer, ok := cn.Peers[srv.PeerName]
		if !ok {
			return fmt.Errorf("net %s peer %s no exist :(", srv.NetworkName, srv.PeerName)
		}
		if !central.AllowedByAny(srv, peer.AllowedSRVs) {
			return fmt.Errorf("srv %d: not allowed", srvI)
		}
		srvs := make([]central.SRV, len(q.SRVs))
		for srvI, srv := range q.SRVs {
			srvs[srvI] = srv.SRV
		}
		var updated bool
		peer.SRVs, updated = central.UpdateSRVs(peer.SRVs, srvs)
		if changed[srv.NetworkName] == nil {
			changed[srv.NetworkName] = make([]string, 0, 1)
		}
		if updated {
			changed[srv.NetworkName] = append(changed[srv.NetworkName], srv.PeerName)
		}
	}
	c.notify(change{Reason: fmt.Sprintf("srvUpdate %s", ti.Name), Changed: changed})
	return nil
}
