package node

import (
	"github.com/nyiyui/qrystal/central"
)

func (n *Node) ReplaceCC(cc2 *central.Config) {
	n.ccLock.Lock()
	defer n.ccLock.Unlock()
	n.cc = *cc2
}
