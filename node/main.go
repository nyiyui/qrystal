package node

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/central"
)

type NodeConfig struct {
	CC               central.Config
	MioAddr          string
	MioToken         []byte
	HokutoAddr       string
	HokutoToken      []byte
	HokutoDNSAddr    string
	HokutoDNSParent  string
	HokutoUseDNS     bool
	CS               CSConfig
	EndpointOverride string
}

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
	}
	if cfg.HokutoDNSAddr != "" {
		addr, err := net.ResolveUDPAddr("udp", cfg.HokutoDNSAddr)
		if err != nil {
			return nil, fmt.Errorf("hokuto resolve addr: %w", err)
		}
		node.hokutoDNSAddr = *addr
		err = node.hokutoInit(cfg.HokutoDNSParent, cfg.HokutoDNSAddr)
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
	eoState              *eoState
	eoStateLock          sync.Mutex
}
