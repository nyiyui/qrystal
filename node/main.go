package node

import (
	"fmt"
	"sync"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/node/api"
)

type NodeConfig struct {
	CC          central.Config
	MioAddr     string
	MioToken    []byte
	HokutoAddr  string
	HokutoToken []byte
	CS          []CSConfig
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

		cs:     cfg.CS,
		csNets: map[string]int{},
		csCls:  make([]api.CentralSourceClient, len(cfg.CS)),

		mio:    mh,
		hokuto: hh,

		Kiriyama: nil, // set below
	}
	node.Kiriyama = newKiriyama(node)
	for i := 0; i < len(cfg.CS); i++ {
		node.Kiriyama.SetCSReady(i, false)
	}
	return node, nil
}

type Node struct {
	ccLock sync.RWMutex
	cc     central.Config

	cs     []CSConfig
	csNets map[string]int
	csCls  []api.CentralSourceClient

	mio    *mioHandle
	hokuto *mioHandle

	Kiriyama *Kiriyama
}
