package node

import (
	"net"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type CentralConfig struct {
	Networks map[string]*CentralNetwork `yaml:"networks"`
	DialOpts []grpc.DialOption          `yaml:"-"`
}
type CentralNetwork struct {
	name string
	IPs  []IPNet2 `yaml:"ips"`
	// TODO: check if all AllowedIPs are in IPs
	Peers      map[string]*CentralPeer `yaml:"peers"`
	Me         string                  `yaml:"me"`
	Keepalive  time.Duration           `yaml:"keepalive"`
	ListenPort int                     `yaml:"listen-port"`

	lock sync.RWMutex
	// lock is only for myPrivKey.
	myPrivKey *wgtypes.Key
}

type CentralPeer struct {
	name       string
	Host       string   `yaml:"host"`
	AllowedIPs []IPNet2 `yaml:"allowed-ips"`
	// TODO: use UnmarshalYAML
	PublicKey  util.Ed25519PublicKey `yaml:"public-key"`
	CanForward bool                  `yaml:"can-forward"`

	lsa     time.Time
	lsaLock sync.RWMutex

	lock       sync.RWMutex
	latestSync time.Time
	accessible bool
	// accessible represents whether this peer is accessible in the latest sync.
	pubKey *wgtypes.Key
	psk    *wgtypes.Key
}

type IPNet2 struct {
	IPNet net.IPNet
}

func (i *IPNet2) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	err := value.Decode(&raw)
	if err != nil {
		return err
	}
	_, ipNet, err := net.ParseCIDR(raw)
	*i = IPNet2{*ipNet}
	return nil
}

func ToIPNets(is2 []IPNet2) []net.IPNet {
	dest := make([]net.IPNet, len(is2))
	for i, i2 := range is2 {
		dest[i] = net.IPNet(i2.IPNet)
	}
	return dest
}

func FromIPNets(ipNets []net.IPNet) []IPNet2 {
	dest := make([]IPNet2, len(ipNets))
	for i, i2 := range ipNets {
		dest[i] = IPNet2{i2}
	}
	return dest
}
