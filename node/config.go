package node

import (
	"fmt"
	"log"
	"net"

	"github.com/nyiyui/qrystal/mio"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (s *Node) configNetwork(cn *CentralNetwork) (err error) {
	log.Print("preconfig")
	config, err := s.convertNetwork(cn)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	log.Print("postconfig")
	me, ok := cn.Peers[cn.Me]
	if !ok {
		return fmt.Errorf("peer %s not found", cn.Me)
	}
	if me.Host != "" {
		listenPort := cn.ListenPort
		config.ListenPort = &listenPort
	}
	log.Print("preconfigdevice")
	q := mio.ConfigureDeviceQ{
		Name:    cn.name,
		Config:  config,
		Address: ToIPNets(me.AllowedIPs),
	}
	if s.forwardingRequired(cn.name) {
		// TODO: figure out how to run sysctl
		// TODO: how to agree between all peers to select one forwarder? or one forwarder for a specific peer, another forwarder for another peer, and so on?
		outbound, err := getOutbound()
		if err != nil {
			return fmt.Errorf("getOutbound: %w", err)
		}
		q.PostUp = makePostUp(cn.name, outbound)
		q.PostDown = makePostDown(cn.name, outbound)
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

func (s *Node) convertNetwork(cn *CentralNetwork) (config *wgtypes.Config, err error) {
	log.Print("postconfig lock", len(cn.Peers))
	configs := make([]wgtypes.PeerConfig, 0, len(cn.Peers))
	for pn, peer := range cn.Peers {
		log.Printf("peer %s config", pn)
		config, accessible, err := s.convertPeer(cn, peer)
		if err != nil {
			return nil, fmt.Errorf("peer %s: %w", pn, err)
		}
		if accessible {
			configs = append(configs, *config)
		}
	}
	config = &wgtypes.Config{
		PrivateKey:   cn.myPrivKey,
		ReplacePeers: true,
		Peers:        configs,
	}
	return config, nil
}

func (s *Node) convertPeer(cn *CentralNetwork, peer *CentralPeer) (config *wgtypes.PeerConfig, accessible bool, err error) {
	peer.lock.RLock()
	log.Printf("LOCK net %s peer %s", cn.name, peer.name)
	defer peer.lock.RUnlock()
	log.Printf("convertPeer postlock")
	if !peer.accessible {
		return nil, false, nil
	}
	var host *net.UDPAddr
	if peer.Host != "" {
		hostOnly, _, err := net.SplitHostPort(peer.Host)
		if err != nil {
			return nil, false, fmt.Errorf("peer %s: splitting failed", peer.name)
		}
		toResolve := fmt.Sprintf("%s:%d", hostOnly, cn.ListenPort)
		host, err = net.ResolveUDPAddr("udp", toResolve)
		if err != nil {
			return nil, false, fmt.Errorf("peer %s: resolving failed", toResolve)
		}
	}

	if peer.pubKey == nil {
		panic(fmt.Sprintf("net %#v peer %#v pubKey is nil", cn, peer))
	}

	keepalive := cn.Keepalive

	allowedIPs := make([]IPNet2, len(peer.AllowedIPs))
	copy(allowedIPs, peer.AllowedIPs)
	log.Printf("conv net %s peer %s forwards for %s", cn.name, peer.name, peer.ForwardingPeers)
	for _, forwardingPeerName := range peer.ForwardingPeers {
		forwardingPeer := cn.Peers[forwardingPeerName]
		allowedIPs = append(allowedIPs, forwardingPeer.AllowedIPs...)
	}

	return &wgtypes.PeerConfig{
		PublicKey:                   *peer.pubKey,
		Remove:                      false,
		UpdateOnly:                  false,
		PresharedKey:                peer.psk,
		Endpoint:                    host,
		PersistentKeepaliveInterval: &keepalive,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  ToIPNets(allowedIPs),
	}, true, nil

}
