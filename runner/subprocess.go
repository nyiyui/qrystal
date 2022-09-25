//go:build linux

package runner

import (
	"errors"
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
	cmd := exec.Command(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid:         cfg.Credential.UID,
			Gid:         cfg.Credential.GID,
			Groups:      cfg.Credential.Groups,
			NoSetGroups: cfg.Credential.NoSetGroups,
		},
	}
	return cmd, nil
}
