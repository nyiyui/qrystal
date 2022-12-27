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
	"gopkg.in/yaml.v3"
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
	cl.Handle("push", func(cl *rpc2.Client, q *api.PushQ, s *api.PushS) error {
		n.ccLock.Lock()
		defer n.ccLock.Unlock()
		i := q.I
		cc := q.CC
		csc := n.cs[i]

		ccy, _ := yaml.Marshal(cc)
		util.S.Infof("新たなCCを受信: %s", ccy)

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
		util.S.Info("NetworksAllowed")

		for cnn := range cc.Networks {
			n.csNets[cnn] = i
		}
		toRemove := cs.MissingFromFirst(cc.Networks, n.cc.Networks)
		util.S.Info("removeDevices")
		err := n.removeDevices(toRemove)
		if err != nil {
			return fmt.Errorf("rm devs: %w", err)
		}
		util.S.Infof("apply and reify")
		n.applyCC(&cc)
		n.reify()
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
	return cl, conn, nil
}
