package node

import (
	"crypto/tls"
	"fmt"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/cs"
	"github.com/nyiyui/qrystal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (n *Node) setupClient(cl *rpc2.Client) {
	cl.Handle("push", func(cl *rpc2.Client, q *api.PushQ, s *api.PushS) error {
		var err error
		n.ccLock.Lock()
		defer n.ccLock.Unlock()
		i := q.I
		cc := q.CC
		csc := n.cs[i]

		for cnn, cn := range cc.Networks {
			util.S.Debugf("新たなCCを受信: net %s: %s", cnn, cn)
		}

		// NetworksAllowed
		cc2 := map[string]*central.Network{}
		for cnn, cn := range cc.Networks {
			if csc.netAllowed(cnn) {
				cc2[cnn] = cn
			} else {
				util.S.Warnf("net not allowed; ignoring: %s", cnn)
			}
		}
		cc.Networks = cc2

		for cnn := range cc.Networks {
			n.csNets[cnn] = i
		}
		toRemove := cs.MissingFromFirst(cc.Networks, n.cc.Networks)
		n.Kiriyama.SetCSReady(i, false)
		err = n.removeDevices(toRemove)
		if err != nil {
			return fmt.Errorf("rm devs: %w", err)
		}
		n.applyCC(&cc)
		for cnn, cn := range cc.Networks {
			util.S.Debugf("after applyCC: net %s: %s", cnn, cn)
		}
		s.PubKeys = map[string]wgtypes.Key{}
		for cnn, cn := range n.cc.Networks {
			if cn.MyPrivKey == nil {
				key, err := wgtypes.GeneratePrivateKey()
				if err != nil {
					return fmt.Errorf("%d: GeneratePrivateKey: %w", i, err)
				}
				cn.MyPrivKey = &key
				n.cc.Networks[cnn] = cn
			}
			s.PubKeys[cnn] = cn.MyPrivKey.PublicKey()
			util.S.Infof("net %s: my PublicKey is %s", cnn, s.PubKeys[cnn])
		}
		err = n.reify()
		if err != nil {
			return fmt.Errorf("reify: %w", err)
		}
		n.Kiriyama.SetCSReady(i, true)
		return nil
	})
	go cl.Run()
}

func (n *Node) newClient(i int) (*rpc2.Client, *tls.Conn, error) {
	csc := n.cs[i]
	var tlsCfg *tls.Config
	if csc.TLSConfig != nil {
		tlsCfg = csc.TLSConfig.Clone()
	}
	conn, err := tls.Dial("tcp", csc.Host, tlsCfg)
	if err != nil {
		return nil, nil, err
	}
	cl := rpc2.NewClient(conn)
	n.setupClient(cl)
	var b bool
	err = cl.Call("ping", true, &b)
	if err != nil {
		return cl, conn, fmt.Errorf("ping: %s", err)
	}
	return cl, conn, nil
}
