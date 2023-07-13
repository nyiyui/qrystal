package node

import (
	"encoding/json"
	"os"

	"github.com/nyiyui/qrystal/central"
)

type Backport struct {
	CC *central.Config `json:"cc"`
}

// LoadBackport reads a backport from the path and applies it if CC exists in the backport.
// This must not be called after Node.ListenCS.
func (n *Node) LoadBackport() error {
	encoded, err := os.ReadFile(n.backportPath)
	if err != nil {
		return err
	}
	var b Backport
	err = json.Unmarshal(encoded, &b)
	if err != nil {
		return err
	}
	if b.CC != nil {
		func() {
			n.ccLock.Lock()
			defer n.ccLock.Unlock()
			n.cc = *b.CC
		}()
		err = n.update()
		if err != nil {
			return err
		}
	}
	return nil
}

// saveBackport saves a backport from the current CC.
// Note: Node.ccLock must be read-locked (i.e. RLock was called).
func (n *Node) saveBackport() error {
	encoded, err := json.Marshal(Backport{
		CC: &n.cc,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(n.backportPath, encoded, 0o0600)
	if err != nil {
		return err
	}
	return nil
}
