package node

import (
	"crypto/tls"
	"errors"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (n *Node) setupClient(cl *rpc2.Client) {
	cl.Handle("generate", func(cl *rpc2.Client, q *api.GenerateQ, s *api.GenerateS) error {
		n.ccLock.Lock()
		defer n.ccLock.Unlock()
		cn, ok := n.cc.Networks[q.CNN]
		if !ok {
			return errors.New("unknown CN")
		}
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return err
		}
		cn.MyPrivKey = &key
		n.cc.Networks[q.CNN] = cn
		s.PubKey = key
		return nil
	})
	go cl.Run()
}

func (n *Node) newClient(i int) (*rpc2.Client, error) {
	csc := n.cs[i]
	conn, err := tls.Dial("tcp", csc.Host, csc.NewTLSConfig())
	if err != nil {
		return nil, err
	}
	cl := rpc2.NewClient(conn)
	n.setupClient(cl)
	return cl, nil
}
