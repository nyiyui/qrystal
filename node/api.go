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
	cl.Handle("ping", func(cl *rpc2.Client, q *bool, s *bool) error {
		*s = true
		panic("test")
		return nil
	})
	cl.Handle("push", func(cl *rpc2.Client, q *api.PushQ, s *api.PushS) error {
		var err error
		n.ccLock.Lock()
		defer n.ccLock.Unlock()
		i := q.I
		cc := q.CC
		csc := n.cs[i]

		util.S.Debugf("%d CNs received", len(cc.Networks))
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
		n.csReady(i, false)
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
				util.S.Infof("net %s: my *new* PublicKey is %s", cnn, key.PublicKey())
			}
			s.PubKeys[cnn] = cn.MyPrivKey.PublicKey()
		}
		err = n.reify()
		if err != nil {
			return fmt.Errorf("reify: %w", err)
		}
		err = n.updateHokutoCC()
		if err != nil {
			return fmt.Errorf("updateHokutoCC: %w", err)
		}
		n.csReady(i, true)
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
		err = fmt.Errorf("dial: %w", err)
		return nil, nil, err
	}
	cl := rpc2.NewClient(conn)
	n.setupClient(cl)
	var b bool
	err = cl.Call("ping", true, &b)
	if err != nil {
		err = fmt.Errorf("ping: %w", err)
		return cl, conn, fmt.Errorf("ping: %s", err)
	}
	return cl, conn, nil
}
