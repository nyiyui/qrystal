package cs

import (
	"fmt"
	"strings"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
)

func (c *CentralSource) azusa(cl *rpc2.Client, q *api.AzusaQ, s *api.AzusaS) error {
	ti, ok, err := c.Tokens.getToken(q.CentralToken)
	if err != nil {
		return err
	}
	if !ok {
		return newTokenAuthError(q.CentralToken)
	}
	var desc strings.Builder
	for cnn, peer := range q.Networks {
		if !ti.CanPush.Any {
			cpn, ok := ti.CanPush.Networks[cnn]
			if !ok {
				return fmt.Errorf("token %s cannot push to net %s", ti.Name, cnn)
			}
			if peer.Name != cpn.Name {
				return fmt.Errorf("token %s cannot push to net %s with name not %s", ti.Name, cnn, cpn.Name)
			}
			if cpn.CanSeeElement != nil {
				if peer.CanSee == nil || peer.CanSee.Only == nil {
					return fmt.Errorf("token %s cannot push to net %s as peer violates CanSeeElement any", ti.Name, cnn)
				} else if len(MissingFromFirst(SliceToMap(cpn.CanSeeElement), SliceToMap(peer.CanSee.Only))) != 0 {
					return fmt.Errorf("token %s cannot push to net %s as peer violates CanSeeElement %s", ti.Name, cnn, cpn.CanSeeElement)
				}
			}
		}
		_, ok := c.cc.Networks[cnn]
		if !ok {
			return fmt.Errorf("net %s no exist :(", cnn)
		}
		fmt.Fprintf(&desc, "\n- net %s peer %s", cnn, peer.Name)
	}
	util.S.Infof("azusa from token %s to push\n%s", ti.Name, desc)
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
	c.ccLock.Lock()
	defer c.ccLock.Unlock()
	for cnn, peer := range q.Networks {
		cn := c.cc.Networks[cnn]
		cn.Peers[peer.Name] = &central.Peer{
			Name:       peer.Name,
			Host:       peer.Host,
			AllowedIPs: peer.AllowedIPs,
			CanSee:     peer.CanSee,
			Internal:   new(central.PeerInternal),
		}
	}
	return nil
}
