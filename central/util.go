package central

import (
	"fmt"
	"net"

	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
)

func FromIPNetsToAPI(nets []net.IPNet) (dest []*api.IPNet) {
	dest = make([]*api.IPNet, len(nets))
	for i, n := range nets {
		if n.String() == "<nil>" {
			panic("nil IPNet")
		}
		dest[i] = &api.IPNet{Cidr: n.String()}
	}
	return
}

func FromAPIToIPNets(nets []*api.IPNet) (dest []net.IPNet, err error) {
	dest = make([]net.IPNet, len(nets))
	for i, n := range nets {
		dest[i], err = util.ParseCIDR(n.Cidr)
		if err != nil {
			return nil, err
		}
	}
	return
}

func (cn *Network) AssignAddr() (n net.IPNet, err error) {
	var ip net.IP
	for _, ipNet := range cn.IPs {
		var usedIPs []net.IPNet
		for _, peer := range cn.Peers {
			usedIPs = append(usedIPs, ToIPNets(peer.AllowedIPs)...)
		}
		ip, err = util.AssignAddress((*net.IPNet)(&ipNet), usedIPs)
		if err != nil {
			return
		}
		if ip != nil {
			break
		}
	}
	if ip == nil {
		panic(fmt.Sprintf("AssignAddr(%#v): ip is nil", cn))
	}
	return net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(32, 8*net.IPv4len),
	}, nil
}
