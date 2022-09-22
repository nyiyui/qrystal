package node

import (
	"context"
	"fmt"
	"time"

	"github.com/nyiyui/qanms/node/api"
	"google.golang.org/grpc"
)

func (n *Node) setupCS() (api.CentralSourceClient, error) {
	conn, err := grpc.Dial(n.csHost, grpc.WithTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}
	cl := api.NewCentralSourceClient(conn)
	return cl, nil
}
func (n *Node) listenCS() error {
	cl, err := n.setupCS()
	if err != nil {
		return err
	}
	conn, err := cl.Pull(context.Background(), &api.PullQ{
		CentralToken: n.csToken,
	})
	if err != nil {
		return fmt.Errorf("pull init: %w", err)
	}
	for {
		s, err := conn.Recv()
		if err != nil {
			return fmt.Errorf("pull recv: %w", err)
		}
		cc, err := newCCFromAPI(s.Cc)
		if err != nil {
			return fmt.Errorf("conv: %w", err)
		}
		err = func() error {
			n.ccLock.Lock()
			defer n.ccLock.Unlock()
			n.cc = *cc
			res, err := n.Sync(context.Background())
			if err != nil {
				return fmt.Errorf("sync: %w", err)
			}
			// TODO: check res
			// TODO: fallback to previous if all fails? perhaps as an option in PullS?
			_ = res
			return nil
		}()
		if err != nil {
			return err
		}
	}
}
