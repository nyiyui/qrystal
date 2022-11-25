// Package central contains Central configuration for Nodes and CSes.
package central

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v3"
)

// Config is the root.
type Config struct {
	Networks map[string]*Network `yaml:"networks"`
}

// Network configures a CN.
type Network struct {
	Name       string
	IPs        []IPNet          `yaml:"ips"`
	Peers      map[string]*Peer `yaml:"peers"`
	Me         string           `yaml:"me"`
	Keepalive  time.Duration    `yaml:"keepalive"`
	ListenPort int              `yaml:"listen-port"`

	Lock sync.RWMutex
	// lock is only for myPrivKey.
	MyPrivKey *wgtypes.Key
}

// Peer configures a peer.
type Peer struct {
	Name            string
	Host            string                `yaml:"host"`
	AllowedIPs      []IPNet               `yaml:"allowed-ips"`
	ForwardingPeers []string              `yaml:"forwarding-peers"`
	PublicKey       util.Ed25519PublicKey `yaml:"public-key"`
	CanSee          *CanSee               `yaml:"can-see"`
	// If CanSee is nil, this Peer can see all peers.

	Internal *PeerInternal `yaml:"-"`
}

func NewPeerFromAPI(pn string, peer *api.CentralPeer) (peer2 *Peer, err error) {
	if len(peer.PublicKey.Raw) == 0 {
		return nil, errors.New("public key blank")
	}
	if len(peer.PublicKey.Raw) != ed25519.PublicKeySize {
		return nil, errors.New("public key size invalid")
	}
	ipNets, err := FromAPIToIPNets(peer.AllowedIPs)
	if err != nil {
		return nil, fmt.Errorf("ToIPNets: %w", err)
	}
	return &Peer{
		Name:            pn,
		Host:            peer.Host,
		AllowedIPs:      FromIPNets(ipNets),
		ForwardingPeers: peer.ForwardingPeers,
		PublicKey:       util.Ed25519PublicKey(peer.PublicKey.Raw),
		CanSee:          NewCanSeeFromAPI(peer.CanSee),
		Internal:        new(PeerInternal),
	}, nil
}

type PeerInternal struct {
	LSA     time.Time
	LSALock sync.RWMutex

	Lock       sync.RWMutex
	LatestSync time.Time
	Accessible bool
	// accessible represents whether this peer is accessible in the latest sync.
	PubKey *wgtypes.Key
	PSK    *wgtypes.Key
	Creds  credentials.TransportCredentials
	// creds for this specific peer.
}

func (p *Peer) ToAPI() *api.CentralPeer {
	return &api.CentralPeer{
		Host:            p.Host,
		AllowedIPs:      FromIPNetsToAPI(ToIPNets(p.AllowedIPs)),
		ForwardingPeers: p.ForwardingPeers,
		PublicKey:       p.PublicKey.ToAPI(),
		CanSee:          p.CanSee.ToAPI(),
	}
}

type CanSee struct {
	Only []string `yaml:"only"`
}

func NewCanSeeFromAPI(c2 *api.CanSee) *CanSee {
	return &CanSee{Only: c2.Only}
}

func (c *CanSee) ToAPI() *api.CanSee {
	return &api.CanSee{Only: c.Only}
}

// IPNet is a YAML-friendly net.IPNet.
// TODO: move to package util
type IPNet net.IPNet

// UnmarshalYAML implements yaml.Unmarshaler.
func (i *IPNet) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	net, err := util.ParseCIDR(raw)
	*i = IPNet(net)
	return err
}

// MarshalYAML implements yaml.Marshaler.
func (i IPNet) MarshalYAML() (interface{}, error) {
	i2 := net.IPNet(i)
	return i2.String(), nil
}

// ToIPNets converts IPNet slices.
func ToIPNets(is2 []IPNet) []net.IPNet {
	dest := make([]net.IPNet, len(is2))
	for i, i2 := range is2 {
		dest[i] = net.IPNet(i2)
	}
	return dest
}

// ToIPNets converts IPNet slices.
func FromIPNets(ipNets []net.IPNet) []IPNet {
	dest := make([]IPNet, len(ipNets))
	for i, i2 := range ipNets {
		dest[i] = IPNet(i2)
	}
	return dest
}
