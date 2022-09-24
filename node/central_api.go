package node

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"net"

	"github.com/nyiyui/qanms/node/api"
	"github.com/nyiyui/qanms/util"
)

func NewCCFromAPI(cc *api.CentralConfig) (cc2 *CentralConfig, err error) {
	return newCCFromAPI(cc)
}

func newCCFromAPI(cc *api.CentralConfig) (cc2 *CentralConfig, err error) {
	networks := map[string]*CentralNetwork{}
	for key, network := range cc.Networks {
		networks[key], err = newCNFromAPI(key, network)
		if err != nil {
			return nil, fmt.Errorf("net %s: %w", key, err)
		}
	}
	return &CentralConfig{
		Networks: networks,
	}, nil
}
func newCNFromAPI(cnn string, cn *api.CentralNetwork) (cn2 *CentralNetwork, err error) {
	peers := map[string]*CentralPeer{}
	for key, network := range cn.Peers {
		peers[key], err = newPeerFromAPI(key, network)
		if err != nil {
			return nil, fmt.Errorf("peer %s: %w", key, err)
		}
	}
	ips, err := FromAPIToIPNets(cn.Ips)
	if err != nil {
		return nil, err
	}
	return &CentralNetwork{
		name:       cnn,
		IPs:        FromIPNets(ips),
		Peers:      peers,
		Me:         cn.Me,
		Keepalive:  cn.Keepalive.AsDuration(),
		ListenPort: int(cn.ListenPort),
	}, nil
}
func FromAPIToIPNets(nets []*api.IPNet) (dest []net.IPNet, err error) {
	dest = make([]net.IPNet, len(nets))
	var n2 *net.IPNet
	for i, n := range nets {
		_, n2, err = net.ParseCIDR(n.Cidr)
		if err != nil {
			return nil, err
		}
		dest[i] = *n2
	}
	return
}

func newPeerFromAPI(pn string, peer *api.CentralPeer) (peer2 *CentralPeer, err error) {
	if len(peer.PublicKey.Raw) == 0 {
		return nil, errors.New("public key blank")
	}
	if len(peer.PublicKey.Raw) != ed25519.PublicKeySize {
		return nil, errors.New("public key size invalid")
	}
	ipNets, err := FromAPIToIPNets(peer.AllowedIPs)
	if err != nil {
		return nil, fmt.Errorf("ToIPNets: %w", err)
	}
	return &CentralPeer{
		name:       pn,
		Host:       peer.Host,
		AllowedIPs: FromIPNets(ipNets),
		PublicKey:  util.Ed25519PublicKey(peer.PublicKey.Raw),
	}, nil
}
