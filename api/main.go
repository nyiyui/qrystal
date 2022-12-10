package api

import (
	"github.com/nyiyui/qrystal/central"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type PullQ struct {
	CentralToken string
}

type PullS struct {
	CC central.Config
}

type GenerateQ struct {
	CNN string
}

type GenerateS struct {
	PubKey wgtypes.Key
}
