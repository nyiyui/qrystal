// Package central contains Central configuration for Nodes and CSes.
package central

import (
	"encoding/json"
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

const (
	DNone    = 0x0
	DNetwork = 0x1
	DIPs     = 0x2
	DPeer    = 0x4
)

// Config is the root.
type Config struct {
	Desynced int
	Networks map[string]*Network `yaml:"networks"`
}

// Network configures a CN.
type Network struct {
	Desynced   int
	Name       string
	IPs        []IPNet          `yaml:"ips"`
	Peers      map[string]*Peer `yaml:"peers"`
	Me         string           `yaml:"me"`
	Keepalive  Duration         `yaml:"keepalive"`
	ListenPort int              `yaml:"listen-port"`

	// lock is only for myPrivKey.
	MyPrivKey *wgtypes.Key
}

// Peer configures a peer.
type Peer struct {
	Desynced        int
	Name            string
	Host            string   `yaml:"host"`
	AllowedIPs      []IPNet  `yaml:"allowed-ips"`
	ForwardingPeers []string `yaml:"forwarding-peers"`
	CanSee          *CanSee  `yaml:"can-see"`
	// If CanSee is nil, this Peer can see all peers.

	Internal *PeerInternal `yaml:"-"`
}

func NewPeerFromAPI(pn string, peer *api.CentralPeer) (peer2 *Peer, err error) {
	ipNets, err := FromAPIToIPNets(peer.AllowedIPs)
	if err != nil {
		return nil, fmt.Errorf("ToIPNets: %w", err)
	}
	return &Peer{
		Name:            pn,
		Host:            peer.Host,
		AllowedIPs:      FromIPNets(ipNets),
		ForwardingPeers: peer.ForwardingPeers,
		CanSee:          NewCanSeeFromAPI(peer.CanSee),
		Internal:        new(PeerInternal),
	}, nil
}

func (p *Peer) Same(p2 *Peer) bool {
	return p.Name == p2.Name && p.Host == p2.Host && Same(p.AllowedIPs, p2.AllowedIPs) && Same3(p.ForwardingPeers, p2.ForwardingPeers) && p.CanSee.Same(p2.CanSee)
}

type PeerInternal struct {
	LSA     time.Time
	LSALock sync.RWMutex

	Lock       sync.RWMutex
	LatestSync time.Time
	Accessible bool
	// accessible represents whether this peer is accessible in the latest sync.
	PubKey *wgtypes.Key
	Creds  credentials.TransportCredentials
	// creds for this specific peer.
}

func (p *Peer) ToAPI() *api.CentralPeer {
	return &api.CentralPeer{
		Host:            p.Host,
		AllowedIPs:      FromIPNetsToAPI(ToIPNets(p.AllowedIPs)),
		ForwardingPeers: p.ForwardingPeers,
		CanSee:          p.CanSee.ToAPI(),
	}
}

type CanSee struct {
	Only []string `yaml:"only"`
}

func NewCanSeeFromAPI(c2 *api.CanSee) *CanSee {
	if c2 == nil {
		return nil
	}
	return &CanSee{Only: c2.Only}
}

func (c *CanSee) ToAPI() *api.CanSee {
	return &api.CanSee{Only: c.Only}
}

func (c *CanSee) Same(c2 *CanSee) bool {
	if c == nil && c2 == nil {
		return true
	}
	if c == nil && c2 != nil {
		return false
	}
	if c2 == nil {
		return false
	}
	return Same3(c.Only, c2.Only)
}

// Duration is a JSON-friendly time.Duration.
type Duration time.Duration

func (d *Duration) UnmarshalJSON(data []byte) error {
	var raw string
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}
	d2, err := time.ParseDuration(raw)
	*d = Duration(d2)
	return err
}

// IPNet is a YAML-friendly net.IPNet.
// TODO: move to package util
type IPNet net.IPNet

// UnmarshalJSON implements yaml.Unmarshaler.
func (i *IPNet) UnmarshalJSON(data []byte) error {
	var cidr string
	err := json.Unmarshal(data, &cidr)
	if err != nil {
		return err
	}
	net, err := util.ParseCIDR(cidr)
	*i = IPNet(net)
	return err
}

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

func Same(a []IPNet, b []IPNet) bool {
	if len(a) != len(b) {
		return false
	}
	for i, a2 := range a {
		b2 := b[i]
		if (*net.IPNet)(&a2).String() == (*net.IPNet)(&b2).String() {
			return false
		}
	}
	return true
}

func Same3(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, a2 := range a {
		b2 := b[i]
		if a2 == b2 {
			return false
		}
	}
	return true
}

func Same2(a map[string]*Peer, b map[string]*Peer) bool {
	if len(a) != len(b) {
		return false
	}
	for i, a2 := range a {
		b2 := b[i]
		if !a2.Same(b2) {
			return false
		}
	}
	return true
}
