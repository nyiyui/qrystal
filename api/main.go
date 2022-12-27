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
	I  int
	CC central.Config
}

type PushS struct{}

type GenerateQ struct {
	CNNs []string
}

type GenerateS struct {
	PubKeys []wgtypes.Key
}
