package mio

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type devConfig struct {
	Address    []net.IPNet
	PrivateKey *wgtypes.Key
	ListenPort uint
	PostUp     string
	PostDown   string
	Peers      []wgtypes.PeerConfig
}

type scriptError struct {
	err     []byte
	wrapped error
}

func (s *scriptError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "script error:\n%s\n", s.wrapped)
	b.Write(s.err)
	return b.String()
}

func devAdd(name string, cfg devConfig) error {
	privateKey := cfg.PrivateKey.String()
	addresses := make([]string, len(cfg.Address))
	for i := range cfg.Address {
		addresses[i] = cfg.Address[i].String()
	}

	after := toAfter(cfg.Peers)

	address := strings.Join(addresses, ", ")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd := exec.Command("/bin/bash", "./dev-add.sh", name, privateKey, address, cfg.PostUp, cfg.PostDown, after, strconv.FormatUint(uint64(cfg.ListenPort), 10))
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	if err != nil {
		return &scriptError{err: errBuf.Bytes(), wrapped: err}
	}
	if outBuf.Len() != 0 {
		util.S.Warnf("dev-add %s out:\n%s", name, outBuf)
	}
	if errBuf.Len() != 0 {
		util.S.Warnf("dev-add %s err:\n%s", name, errBuf)
	}
	return nil
}

func devRemove(name string) error {
	errBuf := new(bytes.Buffer)
	cmd := exec.Command("/bin/bash", "./dev-remove.sh", name)
	cmd.Stderr = errBuf
	err := cmd.Run()
	if err != nil {
		return &scriptError{err: errBuf.Bytes(), wrapped: err}
	}
	if errBuf.Len() != 0 {
		log.Printf("dev-remove %s err:\n%s", name, errBuf)
	}
	return nil
}

func toAfter(peers []wgtypes.PeerConfig) string {
	b := new(strings.Builder)
	for i, peer := range peers {
		fmt.Fprintf(b, "[Peer] # peer %d (NOTE: peers only need the bare for wg-quick to route things)\n", i)
		fmt.Fprintf(b, "PublicKey=%s\n", peer.PublicKey)
		fmt.Fprint(b, "AllowedIPs=")
		for j, ip := range peer.AllowedIPs {
			fmt.Fprintf(b, "%s", &ip)
			if j != len(peer.AllowedIPs)-1 {
				fmt.Fprint(b, ", ")
			}
		}
		fmt.Fprintf(b, "\n")
	}
	return b.String()
}
