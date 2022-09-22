package node

import (
	"errors"
	"time"

	"google.golang.org/grpc"
)

func (n *Node) listenCentral() error {
	conn, err := grpc.Dial(n.centralHost, grpc.WithTimeout(5*time.Second))
	if err != nil {
		return err
	}
	_ = conn
	return errors.New("not implemented")
}
