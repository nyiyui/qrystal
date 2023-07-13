//go:build linux

package runner

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nyiyui/qrystal/runner/config"
)

type nodeHandle struct {
	Cmd  *exec.Cmd
	Port uint16
}

func newNode(cfg *config.Node, mh, hh *mioHandle) (*nodeHandle, error) {
	backportPath := filepath.Join(os.Getenv("STATE_DIRECTORY"), "node-backport.json")
	_, err := os.Stat(backportPath)
	if os.IsNotExist(err) {
		f, err := os.Create(backportPath)
		if err != nil {
			return nil, fmt.Errorf("create backport file: %w", err)
		}
		err = f.Close()
		if err != nil {
			return nil, fmt.Errorf("close created backport file: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("stat backport file: %w", err)
	}
	err = os.Chmod(backportPath, 0o0600)
	if err != nil {
		return nil, fmt.Errorf("chmod created backport file: %w", err)
	}
	cred, err := cfg.Subprocess.Credential.ToCredential()
	if err != nil {
		return nil, fmt.Errorf("backport file: load credential: %w", err)
	}
	err = os.Chown(backportPath, int(cred.Uid), int(cred.Gid))
	if err != nil {
		return nil, fmt.Errorf("chown created backport file: %w", err)
	}

	err = os.Chown(os.Getenv("STATE_DIRECTORY"), int(cred.Uid), int(cred.Gid))
	if err != nil {
		return nil, fmt.Errorf("chown STATE_DIRECTORY: %w", err)
	}

	cmd, err := newSubprocess(cfg.Subprocess)
	if err != nil {
		return nil, fmt.Errorf("subprocess: %w", err)
	}
	cmd.Dir = cfg.Dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = append(cmd.Env, []string{
		// NOTE: this is for cmd/runner-node.go
		fmt.Sprintf("CONFIG_PATH=%s", cfg.ConfigPath),
		fmt.Sprintf("MIO_ADDR=%s", mh.Addr),
		fmt.Sprintf("MIO_TOKEN=%s", mh.TokenBase64),
	}...)
	if hh != nil {
		cmd.Env = append(cmd.Env, []string{
			fmt.Sprintf("HOKUTO_ADDR=%s", hh.Addr),
			fmt.Sprintf("HOKUTO_TOKEN=%s", hh.TokenBase64),
		}...)
	}
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("node: Wait: %s", err)
			if err, ok := err.(*exec.ExitError); ok {
				log.Printf("node: Wait stderr:\n%s", err.Stderr)
			}
			panic("node failed")
		}
	}()
	return &nodeHandle{Cmd: cmd}, nil
}
