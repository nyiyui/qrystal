package mio

import (
	"bytes"
	"fmt"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func wgConfigToString(config *wgtypes.Config) string {
	b := new(strings.Builder)
	fmt.Fprint(b, "[Interface]\n")
	fmt.Fprintf(b, "PrivateKey = %s\n", config.PrivateKey)
	if config.ListenPort == nil {
		fmt.Fprint(b, "ListenPort is not set\n")
	} else {
		fmt.Fprintf(b, "ListenPort = %v\n", *config.ListenPort)
	}
	if config.FirewallMark == nil {
		fmt.Fprint(b, "FirewallMark is not set\n")
	} else {
		fmt.Fprintf(b, "FirewallMark = %v\n", *config.FirewallMark)
	}
	fmt.Fprintf(b, "ReplacePeers = %t\n", config.ReplacePeers)
	for i, peer := range config.Peers {
		fmt.Fprintf(b, "\n[Peer %d]\n", i)
		fmt.Fprintf(b, "PublicKey = %s\n", peer.PublicKey)
		fmt.Fprintf(b, "Remove = %t\n", peer.Remove)
		fmt.Fprintf(b, "UpdateOnly = %t\n", peer.UpdateOnly)
		fmt.Fprintf(b, "PresharedKey = %s\n", peer.PresharedKey)
		fmt.Fprintf(b, "Endpoint = %s\n", peer.Endpoint)
		fmt.Fprintf(b, "PersistentKeepalive = %s\n", peer.PersistentKeepaliveInterval)
		fmt.Fprintf(b, "ReplaceAllowedIPs = %t\n", peer.ReplaceAllowedIPs)
		allowedIPs := new(bytes.Buffer)
		for i, allowedIP := range peer.AllowedIPs {
			fmt.Fprintf(allowedIPs, "%s", allowedIP)
			if i != len(peer.AllowedIPs)-1 {
				fmt.Fprint(allowedIPs, ", ")
			}
		}
		fmt.Fprintf(b, "AllowedIPs = %s\n", allowedIPs)
	}
	return b.String()
}

func (sm *Mio) ensureTokenOk(token []byte) {
	if !bytes.Equal(sm.token, token) {
		// Better to die as *something* is broken or someone is trying to do something bad.
		panic("token mismatch")
	}

}
