//go:build linux

package runner

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/nyiyui/qanms/runner/config"
)

// mioHandle has information about a Mio process.
type mioHandle struct {
	Port        uint16
	Token       []byte
	TokenBase64 string
	Cmd         *exec.Cmd
}

func newMio(cfg *config.Mio) (*mioHandle, error) {
	cmd, err := newSubprocess(cfg.Subprocess)
	if err != nil {
		return nil, fmt.Errorf("subprocess: %w", err)
	}
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("pipe: %w", err)
	}
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	reader := bufio.NewReader(stdout)

	portRaw, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read port: %w", err)
	}
	if !strings.HasPrefix(portRaw, "port:") {
		return nil, fmt.Errorf("rawPort doesn't have prefix: %s", strconv.Quote(portRaw))
	}
	portRaw = strings.TrimSpace(portRaw[5:])
	port, err := strconv.ParseUint(portRaw, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("parse port: %w", err)
	}

	tokenRaw, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read token: %w", err)
	}
	if !strings.HasPrefix(tokenRaw, "token:") {
		return nil, fmt.Errorf("rawPort doesn't have prefix: %s", strconv.Quote(tokenRaw))
	}
	tokenRaw = strings.TrimSpace(tokenRaw[6:])
	token, err := base64.StdEncoding.DecodeString(tokenRaw)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	return &mioHandle{
		Port:        uint16(port),
		Token:       token,
		TokenBase64: tokenRaw,
		Cmd:         cmd,
	}, nil
}
