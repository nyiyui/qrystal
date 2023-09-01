package cs

import (
	"fmt"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
	"golang.org/x/exp/slices"
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
		peer, ok := cn.Peers[srv.PeerName]
		if !ok {
			return fmt.Errorf("net %s peer %s no exist :(", srv.NetworkName, srv.PeerName)
		}
		err = checkPeer(ti, srv.NetworkName, *peer)
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
		i := slices.IndexFunc(peer.SRVs, func(s central.SRV) bool { return s.Service == srv.Service })
		if srv.Service == "" && i != -1 {
			peer.SRVs = append(peer.SRVs[:i], peer.SRVs[i+1:]...)
		} else if i == -1 {
			i = len(peer.SRVs)
			peer.SRVs = append(peer.SRVs, srv.SRV)
		} else {
			peer.SRVs[i] = srv.SRV
		}
	}
	return nil
}
