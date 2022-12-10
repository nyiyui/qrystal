package cs

import (
	"fmt"
	"log"

	"github.com/nyiyui/qrystal/central"
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
			panic(fmt.Sprintf("net %s: token's peer %s doesn't is exist", cnn, me))
		}
		peers := map[string]*central.Peer{}
		for pn, peer := range cn.Peers {
			if mePeer.CanSee != nil && pn != me {
				log.Printf("peer %s CanSee %v", me, mePeer.CanSee)
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
