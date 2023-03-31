package api

import (
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
)

type HPushQ struct {
	CNN      string
	PeerName string
	Peer     central.Peer
}

type HAddTokenQ struct {
	Overwrite bool
	Hash      *util.TokenHash
	Name      string
	CanPull   map[string]string
	CanPush   map[string]CanNetwork
}

type CanNetwork struct {
	PeerName        string
	CanSeeElementOf []string
}

type HRemoveTokenQ struct {
	Hash *util.TokenHash
}
