package simple

import (
	"net"

	"github.com/nyiyui/qrystal/central"
)

type Config struct {
	Networks map[string]Network
}

type ServiceProtocol struct {
	Service  string
	Protocol string
}

type PeerSRV struct {
	PeerName string
	central.SRV
}

type Network struct {
	PeerIPs map[string][]net.IPNet
	SRVs    map[ServiceProtocol][]PeerSRV
}

func ConvertFromCC(cc *central.Config) Config {
	c := Config{Networks: map[string]Network{}}
	for cnn, cn := range cc.Networks {
		c.Networks[cnn] = convertNetwork(cn)
	}
	return c
}

func convertNetwork(cn *central.Network) Network {
	n := Network{
		PeerIPs: map[string][]net.IPNet{},
		SRVs:    map[ServiceProtocol][]PeerSRV{},
	}
	for pn, peer := range cn.Peers {
		for _, ipNet := range peer.AllowedIPs {
			n.PeerIPs[pn] = append(n.PeerIPs[pn], net.IPNet(ipNet))
		}
		for _, record := range peer.SRVs {
			key := ServiceProtocol{record.Service, record.Protocol}
			n.SRVs[key] = append(n.SRVs[key], PeerSRV{
				PeerName: pn,
				SRV:      record,
			})
		}
	}
	return n
}
