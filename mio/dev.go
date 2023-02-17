package mio

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var CommandWg string
var CommandWgQuick string
var CommandBash string

func init() {
	if c := os.Getenv("MIO_COMMAND_WG"); c != "" {
		CommandWg = c
	}
	if c := os.Getenv("MIO_COMMAND_WG_QUICK"); c != "" {
		CommandWgQuick = c
	}
	if c := os.Getenv("MIO_COMMAND_BASJ"); c != "" {
		CommandBash = c
	}
}

type devConfig struct {
	Address    []net.IPNet
	PrivateKey *wgtypes.Key
	ListenPort uint
	DNS        net.UDPAddr
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
	util.S.Infof("devAdd %s", name)
	privateKey := cfg.PrivateKey.String()
	addresses := make([]string, len(cfg.Address))
	for i := range cfg.Address {
		addresses[i] = cfg.Address[i].String()
	}

	after := toAfter(cfg, cfg.Peers)

	address := strings.Join(addresses, ", ")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd := exec.Command(
		CommandBash,
		"./dev-add.sh",
		name,
		privateKey,
		address,
		cfg.PostUp,
		cfg.PostDown,
		after,
		strconv.FormatUint(uint64(cfg.ListenPort), 10),
		CommandWg,
		CommandWgQuick,
	)
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
	util.S.Infof("devRemove %s", name)
	errBuf := new(bytes.Buffer)
	cmd := exec.Command(CommandBash, "./dev-remove.sh", name, CommandWgQuick)
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

func toAfter(cfg devConfig, peers []wgtypes.PeerConfig) string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "DNS=%s\n", cfg.DNS.IP)
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
