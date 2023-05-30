package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nyiyui/qrystal/central"
)

type backport struct {
	CC      central.Config `json:"cc"`
	GenTime time.Time      `json:"genTime"`
}

func (n *Node) loadBackport() (err error) {
	defer func() {
		err = fmt.Errorf("load backport: %w", err)
	}()
	p := filepath.Join(os.Getenv("CACHE_DIRECTORY"), "backport.json")
	var f *os.File
	f, err = os.Open(p)
	if err != nil {
		return
	}
	defer func() {
		err2 := f.Close()
		if err2 != nil {
			err = err2
		}
	}()
	var data backport
	err = json.NewDecoder(f).Decode(&data)
	if err != nil {
		err = fmt.Errorf("decode json: %w", err)
		return
	}

	// apply data
	err = func() error {
		n.ccLock.Lock()
		defer n.ccLock.Unlock()
		if n.cc != nil {
			return errors.New("backport cannot override config (only when cc is nil)")
		}
		n.cc = data.CC
	}()
	if err != nil {
		return
	}
}

func (n *Node) saveBackport() (err error) {
	defer func() {
		err = fmt.Errorf("save backport: %w", err)
	}()
	p := filepath.Join(os.Getenv("CACHE_DIRECTORY"), "backport.json")
	var f *os.File
	f, err = os.Create(p)
	if err != nil {
		return
	}
	defer func() {
		err2 := f.Close()
		if err2 != nil {
			err = err2
		}
	}()
	var data backport
	func() {
		n.ccLock.Lock()
		defer n.ccLock.Unlock()
		data, err = n.genBackport()
		if err != nil {
			err = fmt.Errorf("gen backport: %w", err)
			return
		}
		err = json.NewEncoder(f).Encode(data)
		if err != nil {
			err = fmt.Errorf("encode json: %w", err)
			return
		}
	}()
	return
}

// genBackport generates a backport.
// NOTE: Node.ccLock must be held by callee.
func (n *Node) genBackport() (backport, error) {
	// TODO: validate n.cc for mistakes
	return backport{
		CC:      n.cc,
		GenTime: time.Now(),
	}, nil
}
