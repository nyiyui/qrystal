package node

import (
	"crypto/ed25519"
	"net"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type CentralConfig struct {
	Networks map[string]*CentralNetwork `yaml:"networks"`
	DialOpts []grpc.DialOption
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
	PublicKeyInput string `yaml:"public-key"`
	PublicKey      ed25519.PublicKey

	lock       sync.RWMutex
	latestSync time.Time
	accessible bool
	// accessible represents whether this peer is accessible in the latest sync.
	pubKey *wgtypes.Key
	psk    *wgtypes.Key
}

type IPNet2 struct {
	ipNet net.IPNet
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

func toIPNets(is2 []IPNet2) []net.IPNet {
	dest := make([]net.IPNet, len(is2))
	for i, i2 := range is2 {
		dest[i] = net.IPNet(i2.ipNet)
	}
	return dest
}
