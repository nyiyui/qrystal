package node

import (
	"fmt"
	"net"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
)

func NewCCFromAPI(cc *api.CentralConfig) (cc2 *central.Config, err error) {
	return newCCFromAPI(cc)
}

func newCCFromAPI(cc *api.CentralConfig) (cc2 *central.Config, err error) {
	networks := map[string]*central.Network{}
	for key, network := range cc.Networks {
		networks[key], err = newCNFromAPI(key, network)
		if err != nil {
			return nil, fmt.Errorf("net %s: %w", key, err)
		}
	}
	return &central.Config{
		Networks: networks,
	}, nil
}
func newCNFromAPI(cnn string, cn *api.CentralNetwork) (cn2 *central.Network, err error) {
	peers := map[string]*central.Peer{}
	for key, network := range cn.Peers {
		peers[key], err = central.NewPeerFromAPI(key, network)
		if err != nil {
			return nil, fmt.Errorf("peer %s: %w", key, err)
		}
	}
	ips, err := FromAPIToIPNets(cn.Ips)
	if err != nil {
		return nil, err
	}
	return &central.Network{
		Name:       cnn,
		IPs:        central.FromIPNets(ips),
		Peers:      peers,
		Me:         cn.Me,
		Keepalive:  cn.Keepalive.AsDuration(),
		ListenPort: int(cn.ListenPort),
	}, nil
}

func FromAPIToIPNets(nets []*api.IPNet) (dest []net.IPNet, err error) {
	dest = make([]net.IPNet, len(nets))
	var n2 net.IPNet
	for i, n := range nets {
		n2, err = util.ParseCIDR(n.Cidr)
		if err != nil {
			return nil, err
		}
		dest[i] = n2
	}
	return
}
