package node

import (
	"context"
	"time"

	"github.com/nyiyui/qanms/node/api"
	"google.golang.org/grpc"
)

type clientServer struct {
	cl    api.NodeClient
	token string
}

func (c *Node) ensureClient(ctx context.Context, cnn string, pn string) (err error) {
	csNil := func() bool {
		c.serversLock.RLock()
		defer c.serversLock.RUnlock()
		cs := c.servers[networkPeerPair{cnn, pn}]
		return cs == nil
	}()
	if !csNil {
		return nil
	}
	return c.newClient(ctx, cnn, pn)
}

func (c *Node) newClient(ctx context.Context, cnn string, pn string) (err error) {
	c.serversLock.Lock()
	defer c.serversLock.Unlock()
	peer := c.cc.Networks[cnn].Peers[pn]
	peer.lock.Lock()
	defer peer.lock.Unlock()
	var cs clientServer
	opts := make([]grpc.DialOption, len(c.cc.DialOpts))
	copy(opts, c.cc.DialOpts)
	opts = append(opts, grpc.WithTimeout(5*time.Second))
	conn, err := grpc.DialContext(ctx, peer.Host, opts...)
	if err != nil {
		return err
	}
	cs.cl = api.NewNodeClient(conn)
	c.servers[networkPeerPair{cnn, pn}] = &cs
	return nil
}
