package node

import (
	"fmt"
	"net"

	"github.com/nyiyui/qrystal/mio"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (s *Node) configNetwork(cn *CentralNetwork) (err error) {
	config, err := s.convertNetwork(cn)
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
		Name:    cn.name,
		Config:  config,
		Address: ToIPNets(me.AllowedIPs),
		// TODO: fix to use my IPs
	}
	err = s.mio.ConfigureDevice(q)
	if err != nil {
		return fmt.Errorf("mio: %w", err)
	}
	return nil
}

func (s *Node) convertNetwork(cn *CentralNetwork) (config *wgtypes.Config, err error) {
	cn.lock.RLock()
	defer cn.lock.RUnlock()
	configs := make([]wgtypes.PeerConfig, 0, len(cn.Peers))
	for pn, peer := range cn.Peers {
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
	peer.lock.Lock()
	defer peer.lock.Unlock()
	if !peer.accessible {
		return nil, false, nil
	}
	host, err := net.ResolveUDPAddr("udp", peer.Host)
	if err != nil {
		return nil, false, fmt.Errorf("resolving peer host %s failed", peer.Host)
	}

	if peer.pubKey == nil {
		panic(fmt.Sprintf("net %#v peer %#v pubKey is nil", cn, peer))
	}

	keepalive := cn.Keepalive

	return &wgtypes.PeerConfig{
		PublicKey:                   *peer.pubKey,
		Remove:                      false,
		UpdateOnly:                  false,
		PresharedKey:                peer.psk,
		Endpoint:                    host,
		PersistentKeepaliveInterval: &keepalive,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  ToIPNets(peer.AllowedIPs),
	}, true, nil

}
