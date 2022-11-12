//go:build linux

package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/nyiyui/qrystal/runner/config"
)

func newSubprocess(cfg config.Subprocess) (*exec.Cmd, error) {
	if cfg.Path == "" {
		return nil, errors.New("blank path")
	}
	path, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, err
	}

	cred, err := cfg.Credential.ToCredential()
	if err != nil {
		return nil, fmt.Errorf("ToCredential: %w", err)
	}

	cmd := exec.Command(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: cred,
	}
	cmd.Env = os.Environ()
	return cmd, nil
}
