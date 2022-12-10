package node

import (
	"fmt"
	"net"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (s *Node) reify() (err error) {
	for cnn, cn := range s.cc.Networks {
		err = s.reifyCN(cn)
		if err != nil {
			err = fmt.Errorf("reify net %s: %w", cnn, err)
			return
		}
	}
	return
}

func (s *Node) reifyCN(cn *central.Network) (err error) {
	config, err := s.convCN(cn)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	me, ok := cn.Peers[cn.Me]
	if !ok {
		return fmt.Errorf("peer %s not found", cn.Me)
	}
	if me.Host != "" {
		listenPort := cn.ListenPort
		config.ListenPort = &listenPort
	}
	q := mio.ConfigureDeviceQ{
		Name:    cn.Name,
		Config:  config,
		Address: central.ToIPNets(me.AllowedIPs),
	}
	if s.forwardingRequired(cn.Name) {
		// TODO: figure out how to run sysctl
		// TODO: how to agree between all peers to select one forwarder? or one forwarder for a specific peer, another forwarder for another peer, and so on?
		outbound, err := getOutbound()
		if err != nil {
			return fmt.Errorf("getOutbound: %w", err)
		}
		q.PostUp = makePostUp(cn.Name, outbound)
		q.PostDown = makePostDown(cn.Name, outbound)
		err = s.mio.Forwarding(mio.ForwardingQ{
			Type:   mio.ForwardingTypeIPv4,
			Enable: true,
		})
		if err != nil {
			return fmt.Errorf("Forwarding: %w", err)
		}
	}
	err = s.mio.ConfigureDevice(q)
	if err != nil {
		return fmt.Errorf("mio: %w", err)
	}
	return nil
}

func (s *Node) convCN(cn *central.Network) (config *wgtypes.Config, err error) {
	configs := make([]wgtypes.PeerConfig, 0, len(cn.Peers))
	for pn, peer := range cn.Peers {
		config, accessible, err := s.convPeer(cn, peer)
		if err != nil {
			return nil, fmt.Errorf("peer %s: %w", pn, err)
		}
		if accessible {
			configs = append(configs, *config)
		}
	}
	config = &wgtypes.Config{
		PrivateKey:   cn.MyPrivKey,
		ReplacePeers: true,
		Peers:        configs,
		ListenPort:   &cn.ListenPort,
	}
	return config, nil
}

func (s *Node) convPeer(cn *central.Network, peer *central.Peer) (config *wgtypes.PeerConfig, accessible bool, err error) {
	peer.Internal.Lock.RLock()
	defer peer.Internal.Lock.RUnlock()
	if !peer.Internal.Accessible {
		return nil, false, nil
	}
	var host *net.UDPAddr
	if peer.Host != "" {
		hostOnly, _, err := net.SplitHostPort(peer.Host)
		if err != nil {
			return nil, false, fmt.Errorf("peer %s: splitting failed", peer.Name)
		}
		toResolve := fmt.Sprintf("%s:%d", hostOnly, cn.ListenPort)
		host, err = net.ResolveUDPAddr("udp", toResolve)
		if err != nil {
			return nil, false, fmt.Errorf("peer %s: resolving failed", toResolve)
		}
	}

	if peer.Internal.PubKey == nil {
		panic(fmt.Sprintf("net %#v peer %#v pubKey is nil", cn, peer))
	}

	keepalive := cn.Keepalive

	allowedIPs := make([]central.IPNet, len(peer.AllowedIPs))
	copy(allowedIPs, peer.AllowedIPs)
	util.S.Infof("conv net %s peer %s forwards for %s", cn.Name, peer.Name, peer.ForwardingPeers)
	for _, forwardingPeerName := range peer.ForwardingPeers {
		forwardingPeer := cn.Peers[forwardingPeerName]
		allowedIPs = append(allowedIPs, forwardingPeer.AllowedIPs...)
	}

	return &wgtypes.PeerConfig{
		PublicKey:                   *peer.Internal.PubKey,
		Remove:                      false,
		UpdateOnly:                  false,
		PresharedKey:                peer.Internal.PSK,
		Endpoint:                    host,
		PersistentKeepaliveInterval: &keepalive,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  central.ToIPNets(allowedIPs),
	}, true, nil
}
