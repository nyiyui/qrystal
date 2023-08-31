package api

import (
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type SyncQ struct {
	CentralToken util.Token
}

type SyncS struct{}

type PushQ struct {
	CC central.Config
}

type PushS struct {
	PubKeys map[string]wgtypes.Key
}

type SRVUpdateQ struct {
	CentralToken util.Token
	SRVs         []SRV
}

type SRVUpdateS struct{}

type SRV struct {
	NetworkName string
	PeerName    string

	central.SRV
}

type GenerateQ struct {
	CNNs []string
}

type GenerateS struct {
	PubKeys []wgtypes.Key
}

type AzusaQ struct {
	Networks     map[string]central.Peer
	CentralToken util.Token
}

type AzusaS struct{}
