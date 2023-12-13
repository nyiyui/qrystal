package node

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"

	"github.com/nyiyui/qrystal/util"
)

type eoQ struct {
	CNN      string `json:"cnn"`
	PN       string `json:"pn"`
	Endpoint string `json:"endpoint"`
}

type eoS struct {
	Endpoint string `json:"endpoint"`
}

func (n *Node) getEO(q eoQ) (overriddenEndpoint string, err error) {
	qData, err := json.Marshal(q)
	if err != nil {
		return "", err
	}

	stdoutBuf := new(bytes.Buffer)
	cmd := &exec.Cmd{
		Path:   n.endpointOverridePath,
		Stdin:  bytes.NewBuffer(qData),
		Stdout: stdoutBuf,
		Stderr: os.Stderr,
	}
	// TODO: re-check if piping EO stderr to my stderr is an ok-ish idea
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	var s eoS
	err = json.NewDecoder(stdoutBuf).Decode(&s)
	if err != nil {
		return "", err
	}
	return s.Endpoint, nil
}

// getEOLog returns the overriden endpoint for a peer.
func (n *Node) getEOLog(q eoQ) (overriddenEndpoint string) {
	if n.endpointOverridePath == "" {
		panic("eo not exist")
		return q.Endpoint
	}
	oe, err := n.getEO(q)
	if err != nil {
		util.S.Errorf("endpoint override: get failed, falling back to original address: %s", err)
		return q.Endpoint
	}
	util.S.Infof("endpoint override: override %s with %s", q.Endpoint, oe)
	return oe
}
