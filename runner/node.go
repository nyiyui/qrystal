//go:build linux

package runner

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/nyiyui/qrystal/runner/config"
)

type nodeHandle struct {
	Cmd  *exec.Cmd
	Port uint16
}

func newNode(cfg *config.Node, mh *mioHandle) (*nodeHandle, error) {
	cmd, err := newSubprocess(cfg.Subprocess)
	if err != nil {
		return nil, fmt.Errorf("subprocess: %w", err)
	}
	cmd.Dir = cfg.Dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = []string{
		// NOTE: this is for cmd/runner-node.go
		fmt.Sprintf("CONFIG_PATH=%s", cfg.ConfigPath),
		fmt.Sprintf("MIO_ADDR=%s", mh.Addr),
		fmt.Sprintf("MIO_TOKEN=%s", mh.TokenBase64),
	}
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	return &nodeHandle{Cmd: cmd}, nil
}
