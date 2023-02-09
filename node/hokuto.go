package node

import (
	"fmt"

	"github.com/nyiyui/qrystal/hokuto"
)

// updateHokutoCC updates hokuto's copy of CC.
// NOTE: Node.ccLock must be locked!
func (n *Node) updateHokutoCC() error {
	var dummy bool
	q := hokuto.UpdateCCQ{
		Token: n.hokuto.token,
		CC:    &n.cc,
	}
	err := n.hokuto.client.Call("Hokuto.UpdateCC", q, &dummy)
	if err != nil {
		return fmt.Errorf("call: %w", err)
	}
	return nil
}

func (n *Node) hokutoInit(parent, addr, upstream string) error {
	var dummy bool
	q := hokuto.InitQ{
		Parent:   parent,
		Addr:     addr,
		Upstream: upstream,
	}
	err := n.hokuto.client.Call("Hokuto.Init", q, &dummy)
	if err != nil {
		return fmt.Errorf("call: %w", err)
	}
	return nil
}
