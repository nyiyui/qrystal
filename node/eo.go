package node

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/nyiyui/qrystal/util"
)

type eoState struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	lock   sync.Mutex
}

func (n *Node) initEO() error {
	n.eoStateLock.Lock()
	defer n.eoStateLock.Unlock()
	return n.initEORaw()
}

// initEORaw initializes Node.eoState.
// Node.eoState is not nil if returned error is nil.
func (n *Node) initEORaw() error {
	var err error
	// n.eoStateLock MUST be locked!
	if n.endpointOverridePath == "" {
		return nil
	}
	es := new(eoState)
	es.cmd = &exec.Cmd{
		Path:   n.endpointOverridePath,
		Stderr: os.Stderr,
	}
	es.stdin, err = es.cmd.StdinPipe()
	if err != nil {
		return err
	}
	es.stdout, err = es.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	// TODO: re-check if piping EO stderr to my stderr is an ok-ish idea
	err = es.cmd.Start()
	if err != nil {
		return err
	}
	n.eoState = es
	return nil
}

func (n *Node) deinitEORaw() error {
	// n.eoStateLock MUST be locked!
	err := n.eoState.cmd.Process.Kill()
	if err != nil {
		return err
	}
	n.eoState = nil
	return nil
}

type eoQ struct {
	CNN      string `json:"cnn"`
	PN       string `json:"pn"`
	Endpoint string `json:"endpoint"`
}

type eoS struct {
	Endpoint string `json:"endpoint"`
}

func (n *Node) getEO(q eoQ) (overriddenEndpoint string, err error) {
	// n.eoStateLock MUST be locked!
	// TODO: restart EO if write/read error on stdin/stdout
	n.eoState.lock.Lock()
	defer n.eoState.lock.Unlock()
	err = json.NewEncoder(n.eoState.stdin).Encode(q)
	if err != nil {
		fmt.Fprint(n.eoState.stdin, "\n")
		return "", err
	}
	_, err = fmt.Fprint(n.eoState.stdin, "\n")
	if err != nil {
		return "", err
	}
	data, err := bufio.NewReader(n.eoState.stdout).ReadBytes('\n')
	if err != nil {
		return "", err
	}
	var s eoS
	err = json.Unmarshal(data, &s)
	if err != nil {
		return "", err
	}
	return s.Endpoint, nil
}

// getEOLog returns the overriden endpoint for a peer.
func (n *Node) getEOLog(q eoQ) (overriddenEndpoint string) {
	n.eoStateLock.Lock()
	defer n.eoStateLock.Unlock()
	if n.eoState == nil {
		err := n.initEORaw()
		if err != nil {
			util.S.Errorf("endpoint override: init: %s", err)
			return q.Endpoint
		}
	}
	oe, err := n.getEO(q)
	if err != nil {
		util.S.Errorf("endpoint override: get: %s", err)
		util.S.Info("endpoint override: destroying…")
		err = n.deinitEORaw()
		util.S.Errorf("endpoint override: deinit: %s", err)
		return q.Endpoint
	}
	return oe
}
