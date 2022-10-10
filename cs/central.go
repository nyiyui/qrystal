package cs

import (
	"crypto/ed25519"
	"fmt"
	"log"

	"github.com/nyiyui/qrystal/node"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"google.golang.org/protobuf/types/known/durationpb"
)

func (s *CentralSource) convertCC(tokenNetworks map[string]string) (*api.CentralConfig, error) {
	s.ccLock.RLock()
	defer s.ccLock.RUnlock()
	cc := s.cc
	networks := map[string]*api.CentralNetwork{}
	for cnn, cn := range cc.Networks {
		me, ok := tokenNetworks[cnn]
		if !ok {
			continue
		}
		peers := map[string]*api.CentralPeer{}
		for pn, peer := range cn.Peers {
			log.Printf("net %s peer %s: %#v", cnn, pn, peer)
			peers[pn] = &api.CentralPeer{
				Host:       peer.Host,
				AllowedIPs: FromIPNets(node.ToIPNets(peer.AllowedIPs)),
				PublicKey: &api.PublicKey{
					Raw: []byte(peer.PublicKey),
				},
			}
		}
		networks[cnn] = &api.CentralNetwork{
			Ips:        FromIPNets(node.ToIPNets(cn.IPs)),
			Me:         me,
			Keepalive:  durationpb.New(cn.Keepalive),
			ListenPort: int32(cn.ListenPort),
			Peers:      peers,
		}
	}
	return &api.CentralConfig{
		Networks: networks,
	}, nil
}

func convertPeer(peer *api.CentralPeer) (*node.CentralPeer, error) {
	allowedIPs, err := ToIPNets(peer.AllowedIPs)
	if err != nil {
		return nil, fmt.Errorf("AllowedIPs: %w", err)
	}
	if l := len(peer.PublicKey.Raw); l != ed25519.PublicKeySize {
		return nil, fmt.Errorf("PublicKey: invalid size %d b", l)
	}
	return &node.CentralPeer{
		Host:       peer.Host,
		AllowedIPs: node.FromIPNets(allowedIPs),
		PublicKey:  util.Ed25519PublicKey(peer.PublicKey.Raw),
	}, nil
}
