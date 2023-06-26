package node

import (
	"fmt"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/util"
)

func (n *Node) azusa(networks map[string]central.Peer, cl *rpc2.Client) (err error) {
	util.S.Infof("azusa: call: %s", networks)
	var s api.AzusaS
	err = cl.Call("azusa", &api.AzusaQ{Networks: networks, CentralToken: n.cs.Token}, &s)
	if err != nil {
		err = fmt.Errorf("call: %w", err)
		return
	}
	return
}
