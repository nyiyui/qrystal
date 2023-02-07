package central

import (
	"fmt"
	"net"

	"github.com/nyiyui/qrystal/util"
)

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
