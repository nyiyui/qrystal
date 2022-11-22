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

type Config struct {
	Networks map[string]*Network `yaml:"networks"`
}

type Network struct {
	Name string
	IPs  []IPNet `yaml:"ips"`
	// TODO: check if all AllowedIPs are in IPs
	Peers      map[string]*Peer `yaml:"peers"`
	Me         string           `yaml:"me"`
	Keepalive  time.Duration    `yaml:"keepalive"`
	ListenPort int              `yaml:"listen-port"`

	Lock sync.RWMutex
	// lock is only for myPrivKey.
	MyPrivKey *wgtypes.Key
}

type Peer struct {
	Name            string
	Host            string   `yaml:"host"`
	AllowedIPs      []IPNet  `yaml:"allowed-ips"`
	ForwardingPeers []string `yaml:"forwarding-peers"`
	// TODO: use UnmarshalYAML
	PublicKey  util.Ed25519PublicKey `yaml:"public-key"`
	CanForward bool                  `yaml:"can-forward"`

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

type IPNet net.IPNet

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

func (i IPNet) MarshalYAML() (interface{}, error) {
	i2 := net.IPNet(i)
	return i2.String(), nil
}

func ToIPNets(is2 []IPNet) []net.IPNet {
	dest := make([]net.IPNet, len(is2))
	for i, i2 := range is2 {
		dest[i] = net.IPNet(i2)
	}
	return dest
}

func FromIPNets(ipNets []net.IPNet) []IPNet {
	dest := make([]IPNet, len(ipNets))
	for i, i2 := range ipNets {
		dest[i] = IPNet(i2)
	}
	return dest
}
