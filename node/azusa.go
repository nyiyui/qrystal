package node

import (
	"fmt"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
)

func (n *Node) azusa(networks map[string]central.Peer, cl *rpc2.Client) (err error) {
	var s api.AzusaS
	err = cl.Call("azusa", &api.AzusaQ{Networks: networks, CentralToken: n.cs.Token}, &s)
	if err != nil {
		err = fmt.Errorf("call: %w", err)
		return
	}
	return
}
