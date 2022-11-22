// Package central contains Central configuration for Nodes and CSes.
package central

import (
	"net"
	"sync"
	"time"

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
	CanSee          []string              `yaml:"can-see"`
	// If CanSee is nil, this Peer can see all peers. If CanSee is a zero-length slice, this Peer cannot see any peers.

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
