package node

import (
	"fmt"
	"net"
	"time"

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
	if cn.Desynced == 0 {
		return nil
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
	if s.HokutoUseDNS {
		q.DNS = &s.hokutoDNSAddr
	}
	if s.forwardingRequired(cn.Name) {
		// TODO: figure out how to run sysctl
		// TODO: how to agree between all peers to select one forwarder? or one forwarder for a specific peer, another forwarder for another peer, and so on?
		q.PostUp = makePostUp(cn.Name)
		q.PostDown = makePostDown(cn.Name)
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
	for pn := range cn.Peers {
		if pn == cn.Me {
			continue
		}
		peerConfig, ignore, err := s.convPeer(cn, pn)
		if err != nil {
			return nil, fmt.Errorf("peer %s: %w", pn, err)
		}
		if !ignore {
			configs = append(configs, *peerConfig)
		}
	}
	config = &wgtypes.Config{
		PrivateKey:   cn.MyPrivKey,
		ReplacePeers: true, // (cn.Desynced & central.DIPs)==central.DIPs,
		Peers:        configs,
		ListenPort:   &cn.ListenPort,
	}
	return config, nil
}

func (s *Node) convPeer(cn *central.Network, pn string) (config *wgtypes.PeerConfig, ignore bool, err error) {
	peer := cn.Peers[pn]
	peer.Internal.Lock.RLock()
	defer peer.Internal.Lock.RUnlock()
	var host *net.UDPAddr
	if peer.Host != "" {
		var hostOnly string
		hostOnly, _, err = net.SplitHostPort(peer.Host)
		if err != nil {
			err = fmt.Errorf("peer %s: splitting failed", peer.Name)
			return
		}
		toResolve := fmt.Sprintf("%s:%d", hostOnly, cn.ListenPort)
		host, err = net.ResolveUDPAddr("udp", toResolve)
		if err != nil {
			err = fmt.Errorf("peer %s: resolving failed", toResolve)
			return
		}
	}

	if peer.PubKey == (wgtypes.Key{}) {
		ignore = true
		util.S.Warnf("ignore net %s peer %s as no pubkey", cn.Name, peer.Name)
		return
	}

	keepalive := time.Duration(cn.Keepalive)

	allowedIPs := make([]central.IPNet, len(peer.AllowedIPs))
	copy(allowedIPs, peer.AllowedIPs)
	util.S.Infof("conv net %s peer %s forwards for %s", cn.Name, peer.Name, peer.ForwardingPeers)
	for _, forwardingPeerName := range peer.ForwardingPeers {
		forwardingPeer := cn.Peers[forwardingPeerName]
		allowedIPs = append(allowedIPs, forwardingPeer.AllowedIPs...)
	}

	config = &wgtypes.PeerConfig{
		PublicKey:                   peer.PubKey,
		Remove:                      false,
		UpdateOnly:                  false,
		Endpoint:                    host,
		PersistentKeepaliveInterval: &keepalive,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  central.ToIPNets(allowedIPs),
	}
	return
}
