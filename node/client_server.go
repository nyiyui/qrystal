package node

import (
	"context"
	"time"

	"github.com/nyiyui/qrystal/node/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type clientServer struct {
	cl    api.NodeClient
	token string
}

func (c *Node) ensureClient(ctx context.Context, cnn string, pn string) (err error) {
	c.serversLock.Lock()
	defer c.serversLock.Unlock()
	cs := c.servers[networkPeerPair{cnn, pn}]
	if cs != nil {
		return nil
	}
	return c.newClient(ctx, cnn, pn)
}

func (c *Node) newClient(ctx context.Context, cnn string, pn string) (err error) {
	peer := c.cc.Networks[cnn].Peers[pn]
	peer.lock.Lock()
	defer peer.lock.Unlock()
	var cs clientServer
	var creds credentials.TransportCredentials
	if peer.creds != nil {
		creds = peer.creds
	} else {
		creds = credentials.NewTLS(nil)
	}
	conn, err := grpc.DialContext(ctx, peer.Host, grpc.WithTimeout(5*time.Second), grpc.WithTransportCredentials(creds))
	if err != nil {
		return err
	}
	cs.cl = api.NewNodeClient(conn)
	c.servers[networkPeerPair{cnn, pn}] = &cs
	return nil
}
