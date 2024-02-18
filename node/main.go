package node

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/hokuto"
	"golang.org/x/exp/slices"
)

type NodeConfig struct {
	CC                 central.Config
	MioAddr            string
	MioToken           []byte
	HokutoAddr         string
	HokutoToken        []byte
	HokutoDNSAddr      string
	HokutoDNSParent    string
	HokutoUseDNS       bool
	HokutoExtraParents []hokuto.ExtraParent
	CS                 CSConfig
	EndpointOverride   string
	BackportPath       string
	SRVList            string
}

// There must be only one Node instance as a Node can trigger a trace to stop.
func NewNode(cfg NodeConfig) (*Node, error) {
	mh, err := newMio(cfg.MioAddr, cfg.MioToken)
	if err != nil {
		return nil, fmt.Errorf("new mio: %w", err)
	}
	var hh *mioHandle
	if cfg.HokutoAddr != "" {
		hh, err = newMio(cfg.HokutoAddr, cfg.HokutoToken)
		if err != nil {
			return nil, fmt.Errorf("new hokuto: %w", err)
		}
	}

	node := &Node{
		cc: cfg.CC,

		cs: cfg.CS,

		mio:                  mh,
		hokuto:               hh,
		endpointOverridePath: cfg.EndpointOverride,
		backportPath:         cfg.BackportPath,
		srvListPath:          cfg.SRVList,
	}
	if cfg.HokutoDNSAddr != "" {
		addr, err := net.ResolveUDPAddr("udp", cfg.HokutoDNSAddr)
		if err != nil {
			return nil, fmt.Errorf("hokuto resolve addr: %w", err)
		}
		node.hokutoDNSAddr = *addr
		err = node.hokutoInit(cfg.HokutoDNSParent, cfg.HokutoDNSAddr, cfg.HokutoExtraParents)
		if err != nil {
			return nil, fmt.Errorf("hokuto init: %w", err)
		}
	}
	return node, nil
}

type Node struct {
	ccLock sync.RWMutex
	cc     central.Config
	// ccApplyTime is the latest time a backport is updated.
	// This is not meant to
	ccApplyTime time.Time

	ready bool

	cs CSConfig

	mio    *mioHandle
	hokuto *mioHandle

	hokutoDNSAddr net.UDPAddr

	endpointOverridePath string

	backportPath string
	srvListPath  string

	reupdateSRV chan string

	keepaliveEntriesLock sync.Mutex
	keepaliveEntries     []keepaliveEntry // TODO: switch from slice to container/list?
}

type keepaliveEntry struct {
	Secret []byte
	Notify chan<- struct{}
}

func (n *Node) addKeepaliveEntry() (secret []byte, notify <-chan struct{}) {
	n.keepaliveEntriesLock.Lock()
	defer n.keepaliveEntriesLock.Unlock()
	secret = make([]byte, 32) // eh whatever, 32 bytes is probably overkill but who cares if it's a bit slow :)
	_, err := rand.Read(secret)
	if err != nil {
		panic(err)
	}
	notify2 := make(chan struct{}, 1)
	ke := keepaliveEntry{
		Secret: secret,
		Notify: notify2,
	}
	n.keepaliveEntries = append(n.keepaliveEntries, ke)
	return secret, notify2
}

// removeKeepaliveEntry removes the corresponding keepaliveEntry, if it exists. If the corresponding entry does not exist, this function does nothing.
func (n *Node) removeKeepaliveEntry(secret []byte) {
	n.keepaliveEntriesLock.Lock()
	defer n.keepaliveEntriesLock.Unlock()
	i := slices.IndexFunc(n.keepaliveEntries, func(ke keepaliveEntry) bool { return bytes.Equal(ke.Secret, secret) })
	if i == -1 {
		return
	}
	n.keepaliveEntries[i] = keepaliveEntry{} // zero to garbage collect now-unnecessary entry
	slices.Delete(n.keepaliveEntries, i, i+1)
}
