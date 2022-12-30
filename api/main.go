package api

import (
	"github.com/nyiyui/qrystal/central"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type PullQ struct {
	I            int
	CentralToken string
}

type PullS struct{}

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
