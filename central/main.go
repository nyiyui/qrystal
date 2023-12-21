// Package central contains Central configuration for Nodes and CSes.
package central

import (
	"encoding/json"
	"errors"
	"net"
	"time"

	"github.com/nyiyui/qrystal/util"
	"golang.org/x/exp/slices"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gopkg.in/yaml.v3"
)

const (
	DNone    = 0x0
	DNetwork = 0x1
	DIPs     = 0x2
	DKeys    = 0x8
	DPeer    = 0x4
	DSRVs    = 0x10
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
	Desynced int
	// SyncedPeers is for internal use by the cs package.
	// In the cs package, it's used to track which peers have been synced with changes.
	SyncedPeers []string
	Name        string  `yaml:"name" json:"name"`
	Host        string  `yaml:"host" json:"host"`
	AllowedIPs  []IPNet `yaml:"allowedIPs" json:"allowedIPs"`
	CanForward  bool    `yaml:"canForward" json:"canForward"`
	// CanSee determines whether this Peer can see anything (nil) or specfic peers only (non-nil).
	CanSee      *CanSee        `yaml:"canSee" json:"canSee"`
	AllowedSRVs []SRVAllowance `yaml:"allowedSRVs" json:"allowedSRVs"`

	PubKey          wgtypes.Key
	ForwardingPeers []string
	SRVs            []SRV
}

func (p *Peer) String() string {
	out, _ := json.Marshal(p)
	return string(out)
}

func (p *Peer) Same(p2 *Peer) bool {
	return p.Name == p2.Name && p.Host == p2.Host && Same(p.AllowedIPs, p2.AllowedIPs) && Same3(p.ForwardingPeers, p2.ForwardingPeers) && p.CanForward == p2.CanForward && p.CanSee.Same(p2.CanSee) && p.PubKey == p2.PubKey
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

type SRVAllowable interface {
	// AllowedBy returns nil this is allowed by the SRVAllowance, and a non-nil error otherwise.
	AllowedBy(SRVAllowance) error
}

func AllowedByAny(sa SRVAllowable, a2s []SRVAllowance) bool {
	for _, a2 := range a2s {
		err := sa.AllowedBy(a2)
		if err == nil {
			return true
		}
	}
	return false
}

type SRVAllowance struct {
	Service     string `yaml:"service"`
	ServiceAny  bool   `yaml:"serviceAny"`
	PriorityMin uint16 `yaml:"priorityMin"`
	PriorityMax uint16 `yaml:"priorityMax"`
	WeightMin   uint16 `yaml:"weightMin"`
	WeightMax   uint16 `yaml:"weightMax"`
}

func (a SRVAllowance) AllowedBy(a2 SRVAllowance) error {
	if !a2.ServiceAny && a.Service != a2.Service {
		return errors.New("Service mismatch")
	}
	if !(a.PriorityMin >= a2.PriorityMin && a.PriorityMax <= a2.PriorityMax) {
		return errors.New("Priority not in range")
	}
	if !(a.WeightMin >= a2.WeightMin && a.WeightMax <= a2.WeightMax) {
		return errors.New("Weight not in range")
	}
	return nil
}

type SRV struct {
	Service  string
	Protocol string
	Priority uint16
	Weight   uint16
	Port     uint16
}

func (s SRV) AllowedBy(a2 SRVAllowance) error {
	if !a2.ServiceAny && s.Service != a2.Service {
		return errors.New("Service mismatch")
	}
	if !(a2.PriorityMin <= s.Priority && s.Priority <= a2.PriorityMax) {
		return errors.New("Priority not in range")
	}
	if !(a2.WeightMin <= s.Weight && s.Weight <= a2.WeightMax) {
		return errors.New("Weight not in range")
	}
	return nil
}

func UpdateSRVs(target, updater []SRV) (target_ []SRV, updated bool) {
	for _, srv := range updater {
		i := slices.IndexFunc(target, func(s SRV) bool { return s.Service == srv.Service })
		if srv.Service == "" && i != -1 {
			target = append(target[:i], target[i+1:]...)
			updated = true
		} else if i == -1 {
			i = len(target)
			target = append(target, srv)
			updated = true
		} else {
			target[i] = srv
		}
	}
	return target, updated
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

// MarshalJSON implements yaml.Marshaler.
func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(*d).String())
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

// MarshalJSON implements yaml.Marshaler.
func (i IPNet) MarshalJSON() ([]byte, error) {
	i2 := net.IPNet(i)
	return json.Marshal(i2.String())
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

func IPNetSubsetOfAny(subset net.IPNet, supersets []IPNet) bool {
	for _, superset := range supersets {
		if !IPNetSubsetOf(subset, net.IPNet(superset)) {
			return false
		}
	}
	return true
}

func IPNetSubsetOf(subset, superset net.IPNet) bool {
	superMask, _ := superset.Mask.Size()
	subMask, _ := subset.Mask.Size()
	return superMask <= subMask && superset.Contains(subset.IP)
}
