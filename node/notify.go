package node

import "github.com/nyiyui/qrystal/util"

func (n *Node) csReady(i int, ready bool) {
	n.ready[i] = ready
	if ready && n.csAllReady() {
		util.Notify("READY=1")
	}
}

func (n *Node) csAllReady() bool {
	for i := range n.ready {
		if !n.ready[i] {
			return false
		}
	}
	return true
}
