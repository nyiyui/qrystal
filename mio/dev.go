package mio

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type devConfig struct {
	Address    []net.IPNet
	PrivateKey *wgtypes.Key
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
	address := strings.Join(addresses, ", ")
	errBuf := new(bytes.Buffer)
	cmd := exec.Command("/bin/bash", "./dev-add.sh", name, privateKey, address)
	cmd.Stderr = errBuf
	err := cmd.Run()
	if err != nil {
		return &scriptError{err: errBuf.Bytes(), wrapped: err}
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
	return nil
}
