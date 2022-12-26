package node

import (
	"crypto/tls"
	"fmt"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (n *Node) setupClient(cl *rpc2.Client) {
	cl.Handle("generate", func(cl *rpc2.Client, q *api.GenerateQ, s *api.GenerateS) error {
		n.ccLock.Lock()
		defer n.ccLock.Unlock()
		s.PubKeys = make([]wgtypes.Key, len(q.CNNs), len(q.CNNs))
		for i, cnn := range q.CNNs {
			cn, ok := n.cc.Networks[cnn]
			if !ok {
				return fmt.Errorf("%d: unknown CN %s", i, cnn)
			}
			key, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				return fmt.Errorf("%d: GeneratePrivateKey: %w", i, err)
			}
			cn.MyPrivKey = &key
			n.cc.Networks[cnn] = cn
			s.PubKeys[i] = key
		}
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
