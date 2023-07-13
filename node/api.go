package node

import (
	"crypto/tls"
	"fmt"
	"strings"

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
		return nil
	})
	cl.Handle("push", func(cl *rpc2.Client, q *api.PushQ, s *api.PushS) error {
		// TODO: notify ERRNO= for errors?
		var err error
		n.ccLock.Lock()
		defer n.ccLock.Unlock()
		cc := q.CC

		util.Notify(fmt.Sprintf("STATUS=Receiving new CC (%d CNs)...", len(cc.Networks)))
		for cnn, cn := range cc.Networks {
			util.S.Debugf("新たなCCを受信: net %s: %s", cnn, cn)
		}

		// NetworksAllowed
		cc2 := map[string]*central.Network{}
		for cnn, cn := range cc.Networks {
			if n.cs.netAllowed(cnn) {
				cc2[cnn] = cn
			} else {
				util.S.Warnf("net not allowed; ignoring: %s", cnn)
			}
		}
		cc.Networks = cc2

		toRemove := cs.MissingFromFirst(cc.Networks, n.cc.Networks)
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
					return fmt.Errorf("GeneratePrivateKey: %w", err)
				}
				cn.MyPrivKey = &key
				n.cc.Networks[cnn] = cn
				util.S.Infof("net %s: my *new* PublicKey is %s", cnn, key.PublicKey())
			}
			s.PubKeys[cnn] = cn.MyPrivKey.PublicKey()
		}
		util.Notify(fmt.Sprintf("STATUS=Reifying new CC (%d CNs)...", len(cc.Networks)))
		err = n.reify()
		if err != nil {
			return fmt.Errorf("reify: %w", err)
		}
		util.Notify(fmt.Sprintf("STATUS=Updating DNS for new CC (%d CNs)...", len(cc.Networks)))
		err = n.updateHokutoCC()
		if err != nil {
			return fmt.Errorf("updateHokutoCC: %w", err)
		}
		cns := new(strings.Builder)
		for cnn, cn := range n.cc.Networks {
			fmt.Fprintf(cns, " %s", cnn)
		}
		util.Notify(fmt.Sprintf("STATUS=Synced. CNs:%s", cns))
		util.Notify("READY=1")
		n.traceCheck()
		return nil
	})
	go cl.Run()
}

func (n *Node) newClient() (*rpc2.Client, *tls.Conn, error) {
	var tlsCfg *tls.Config
	if n.cs.TLSConfig != nil {
		tlsCfg = n.cs.TLSConfig.Clone()
	}
	conn, err := tls.Dial("tcp", n.cs.Host, tlsCfg)
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
