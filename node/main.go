package node

import (
	"fmt"
	"sync"

	"github.com/nyiyui/qrystal/central"
)

type NodeConfig struct {
	CC                central.Config
	MioAddr           string
	MioToken          []byte
	HokutoAddr        string
	HokutoToken       []byte
	HokutoDNSAddr     string
	HokutoDNSParent   string
	HokutoDNSUpstream string
	CS                []CSConfig
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

		ready: make([]bool, len(cfg.CS)),

		cs:     cfg.CS,
		csNets: map[string]int{},

		mio:    mh,
		hokuto: hh,
	}
	err = node.hokutoInit(cfg.HokutoDNSParent, cfg.HokutoDNSAddr, cfg.HokutoDNSUpstream)
	if err != nil {
		return nil, fmt.Errorf("hokuto init: %w", err)
	}
	return node, nil
}

type Node struct {
	ccLock sync.RWMutex
	cc     central.Config

	ready []bool

	cs     []CSConfig
	csNets map[string]int

	mio    *mioHandle
	hokuto *mioHandle
}
