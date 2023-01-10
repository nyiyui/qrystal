package node

import (
	"fmt"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
)

func (n *Node) azusa(i int, networks map[string]central.Peer, cl *rpc2.Client) (err error) {
	csc := n.cs[i]
	var s api.AzusaS
	n.Kiriyama.SetCS(i, "梓")
	err = cl.Call("azusa", &api.AzusaQ{Networks: networks, CentralToken: csc.Token}, &s)
	if err != nil {
		err = fmt.Errorf("call: %w", err)
		return
	}
	return
}
