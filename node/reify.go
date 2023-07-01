package node

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net"
	"time"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/hokuto"
	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// reify applies current CC to system.
// NOTE: ccLock must be held
func (s *Node) reify() (err error) {
	err = s.reifyHosts()
	if err != nil {
		err = fmt.Errorf("reify hosts: %w", err)
		return
	}
	for cnn, cn := range s.cc.Networks {
		err = s.reifyCN(cn)
		if err != nil {
			err = fmt.Errorf("reify net %s: %w", cnn, err)
			return
		}
	}
	return
}

// NOTE: ccLock must be held
func (n *Node) reifyHosts() (err error) {
	hosts := map[string][]string{}
	for _, cn := range n.cc.Networks {
		for _, peer := range cn.Peers {
			if len(peer.Hosts) > 1 {
				raw := make([]byte, 64)
				_, err := rand.Read(raw)
				if err != nil {
					return err
				}
				key := base32.StdEncoding.EncodeToString(raw)
				hosts[key] = peer.Hosts
				peer.HostsKey = key
			}
		}
	}
	var dummy bool
	q := hokuto.UpdateHostsQ{
		Token: n.hokuto.token,
		Hosts: hosts,
	}
	err = n.hokuto.client.Call("Hokuto.UpdateHosts", q, &dummy)
	if err != nil {
		return fmt.Errorf("hokuto: %w", err)
	}
	return nil
}

// NOTE: ccLock must be held
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
	if len(me.Hosts) != 0 {
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

// NOTE: ccLock must be held
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

// NOTE: ccLock must be held
func (s *Node) convPeer(cn *central.Network, pn string) (config *wgtypes.PeerConfig, ignore bool, err error) {
	peer := cn.Peers[pn]
	endpoint, overridden := s.getEOLog(eoQ{
		CNN:      cn.Name,
		PN:       pn,
		Endpoint: peer.Host,
	})
	if overridden {
		host, err = net.ResolveUDPAddr("udp", endpoint)
		if err != nil {
			err = fmt.Errorf("peer %s: resolving failed", toResolve)
			return
		}
	} else {
		var host *net.UDPAddr
		for _, host := range peer.Hosts {
			var hostOnly string
			hostOnly, _, err = net.SplitHostPort(host)
			if err != nil {
				err = fmt.Errorf("peer %s: splitting failed", peer.Name)
				return
			}
			toResolve := fmt.Sprintf("%s:%d", hostOnly, cn.ListenPort)
			if overridden {
				// if a custom endpoint is given, respect port choices for that
				toResolve = host
			}
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
