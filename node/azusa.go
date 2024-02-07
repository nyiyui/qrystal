package node

import (
	"context"
	"fmt"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
)

func (n *Node) azusa(ctx context.Context, networks map[string]central.Peer, cl *rpc2.Client) (err error) {
	var s api.AzusaS
	err = cl.CallWithContext(ctx, "azusa", &api.AzusaQ{Networks: networks, CentralToken: n.cs.Token}, &s)
	if err != nil {
		err = fmt.Errorf("call: %w", err)
		return
	}
	return
}
