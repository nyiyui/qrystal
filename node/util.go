package node

import (
	"crypto/rand"
	"fmt"

	"github.com/nyiyui/qrystal/central"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func readRand(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func ensureWGPrivKey(cn *central.Network) error {
	if cn.MyPrivKey != nil {
		return nil
	}
	myWGPrivKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return fmt.Errorf("gen privkey: %w", err)
	}
	cn.MyPrivKey = &myWGPrivKey
	return nil
}

type networkPeerPair struct {
	network string
	peer    string
}

func (n *Node) ReplaceCC(cc2 *central.Config) {
	n.ccLock.Lock()
	defer n.ccLock.Unlock()
	n.cc = *cc2
}
