package cs

import (
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
)

func (s *CentralSource) copyCC(tokenNetworks map[string]string) (*central.Config, error) {
	s.ccLock.RLock()
	defer s.ccLock.RUnlock()
	cc := s.cc
	networks := map[string]*central.Network{}
	for cnn, cn := range cc.Networks {
		me, ok := tokenNetworks[cnn]
		if !ok {
			continue
		}
		mePeer := cn.Peers[me]
		if mePeer == nil {
			// continue because the peer might be added later
			util.S.Warnf("net %s: token's peer %s doesn't exist", cnn, me)
			continue
		}
		peers := map[string]*central.Peer{}
		for pn, peer := range cn.Peers {
			if mePeer.CanSee != nil && pn != me {
				if !contains(mePeer.CanSee.Only, pn) {
					continue
				}
			}
			peers[pn] = peer
		}
		cn2 := *cn
		cn2.Me = me
		networks[cnn] = &cn2
	}
	return &central.Config{
		Networks: networks,
	}, nil
}
