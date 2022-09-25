package node

import (
	"crypto/ed25519"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type exchangeReq struct {
	HRPubKey   ed25519.PublicKey `json:"hr-qrystal-pubkey"`
	Challenge  []byte            `json:"chall"`
	HRWGPubKey wgtypes.Key       `json:"hr-wg-pubkey"`
	HRWGPSK    wgtypes.Key       `json:"hr-wg-psk"`
	LRPX       int               `json:"lrpx"`
	LRPN       int               `json:"lrpn"`
}

type exchangeResp struct {
	ChallengeResp     []byte      `json:"chall-resp"`
	ChallengeAppendee []byte      `json:"chall-appendee"`
	LRWGPubKey        wgtypes.Key `json:"lr-wg-pubkey"`
	LRWGPSK           wgtypes.Key `json:"lr-wg-psk"`
}
