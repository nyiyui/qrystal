package api

import (
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type SyncQ struct {
	I            int
	CentralToken util.Token
}

type SyncS struct{}

type PushQ struct {
	I int
	// I is the index of the CS in this specific Node. This is required
	// to get allowed networks etc.
	// TODO: perhaps some way to prevent the CS from spoofing this
	CC central.Config
}

type PushS struct {
	PubKeys map[string]wgtypes.Key
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
