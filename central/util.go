package central

import (
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
