// Package central contains Central configuration for Nodes and CSes.
package central

import (
	"encoding/json"
	"net"
	"time"

	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gopkg.in/yaml.v3"
)

const (
	DNone    = 0x0
	DNetwork = 0x1
	DIPs     = 0x2
	DKeys    = 0x8
	DPeer    = 0x4
	DHosts   = 0x16
)

// Config is the root.
type Config struct {
	Desynced int
	Networks map[string]*Network `yaml:"networks" json:"networks"`
}

// Network configures a CN.
type Network struct {
	Desynced   int
	Name       string
	IPs        []IPNet          `yaml:"ips" json:"ips"`
	Peers      map[string]*Peer `yaml:"peers" json:"peers"`
	Me         string           `yaml:"me" json:"me"`
	Keepalive  Duration         `yaml:"keepalive" json:"keepalive"`
	ListenPort int              `yaml:"listenPort" json:"listenPort"`

	// lock is only for myPrivKey.
	MyPrivKey *wgtypes.Key `json:"myPrivKey"`
}

func (cn *Network) String() string {
	out, _ := json.Marshal(cn)
	return string(out)
}

// Peer configures a peer.
type Peer struct {
	Desynced        int
	Name            string   `yaml:"name" json:"name"`
	Hosts           []string `yaml:"hosts" json:"hosts"`
	AllowedIPs      []IPNet  `yaml:"allowedIPs" json:"allowedIPs"`
	ForwardingPeers []string `yaml:"forwardingPeers" json:"forwardingPeers"`
	// CanSee determines whether this Peer can see anything (nil) or specfic peers only (non-nil).
	// TODO: when CanSee.Only is blank, this is interpreted as nil → no way to distinguish between seeing nothing and everything
	CanSee *CanSee `yaml:"canSee" json:"canSee"`

	PubKey   wgtypes.Key
	HostsKey string
}

func (p *Peer) String() string {
	out, _ := json.Marshal(p)
	return string(out)
}

func (p *Peer) Same(p2 *Peer) bool {
	return p.Name == p2.Name && Same3(p.Hosts, p2.Hosts) && Same(p.AllowedIPs, p2.AllowedIPs) && Same3(p.ForwardingPeers, p2.ForwardingPeers) && p.CanSee.Same(p2.CanSee) && p.PubKey == p2.PubKey
}

type CanSee struct {
	Only []string `yaml:"only"`
	// Only means that this peer can see itself and only the listed peers.
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

// Duration is a encoding-friendly time.Duration.
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler.
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

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	d2, err := time.ParseDuration(raw)
	*d = Duration(d2)
	return err
}

// MarshalYAML implements yaml.Marshaler.
func (d *Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(*d).String(), nil
}

// IPNet is a YAML- and JSON- friendly net.IPNet.
// TODO: move to package util
type IPNet net.IPNet

// UnmarshalJSON implements json.Unmarshaler.
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
