//go:build linux

package runner

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/nyiyui/qrystal/runner/config"
)

// mioHandle has information about a Mio process.
type mioHandle struct {
	Addr        string
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
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("mio: Wait: %s", err)
			if err, ok := err.(*exec.ExitError); ok {
				log.Printf("mio: Wait stderr:\n%s", err.Stderr)
			}
		}
		log.Print("mio exited")
		panic("mio failed")
	}()
	addr, tokenRaw, token, err := readTokenAddr(stdout)
	if err != nil {
		return nil, fmt.Errorf("read token addr: %w", err)
	}

	return &mioHandle{
		Addr:        addr,
		Token:       token,
		TokenBase64: tokenRaw,
		Cmd:         cmd,
	}, nil
}

func readTokenAddr(stdout io.Reader) (addr, tokenBase64 string, token []byte, err error) {
	reader := bufio.NewReader(stdout)

	addrRaw, err := reader.ReadString('\n')
	if err != nil {
		return "", "", nil, fmt.Errorf("read addr: %w", err)
	}
	if !strings.HasPrefix(addrRaw, "addr:") {
		return "", "", nil, fmt.Errorf("rawPort doesn't have prefix: %s", strconv.Quote(addrRaw))
	}
	addr = strings.TrimSpace(addrRaw[5:])
	if err != nil {
		return "", "", nil, fmt.Errorf("parse addr: %w", err)
	}

	tokenRaw, err := reader.ReadString('\n')
	if err != nil {
		return "", "", nil, fmt.Errorf("read token: %w", err)
	}
	if !strings.HasPrefix(tokenRaw, "token:") {
		return "", "", nil, fmt.Errorf("rawToken doesn't have prefix: %s", strconv.Quote(tokenRaw))
	}
	tokenRaw = strings.TrimSpace(tokenRaw[6:])
	token, err = base64.StdEncoding.DecodeString(tokenRaw)
	if err != nil {
		return "", "", nil, fmt.Errorf("parse token: %w", err)
	}
	return addr, tokenRaw, token, nil
}
