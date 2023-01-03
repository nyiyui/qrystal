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

func (cc *Config) Assign() (err error) {
	for cnn, cn := range cc.Networks {
		err := cn.Assign()
		if err != nil {
			return fmt.Errorf("net %s: %w", cnn, err)
		}
	}
	return nil
}

func (cn *Network) Assign() (err error) {
	for pn, peer := range cn.Peers {
		if len(peer.AllowedIPs) == 0 {
			continue
		}
		err := cn.AssignPeer(pn)
		if err != nil {
			return err
		}
	}
	return
}

func (cn *Network) AssignPeer(pn string) (err error) {
	var ip net.IP
	for _, ipNet := range cn.IPs {
		var usedIPs []net.IPNet
		for _, peer := range cn.Peers {
			usedIPs = append(usedIPs, ToIPNets(peer.AllowedIPs)...)
		}
		ip, err = util.AssignAddress((*net.IPNet)(&ipNet), usedIPs)
		if err != nil {
			err = fmt.Errorf("peer %s: %w", pn, err)
			return
		}
	}
	cn.Peers[pn].AllowedIPs = []IPNet{IPNet(net.IPNet{IP: ip})}
	return
}
