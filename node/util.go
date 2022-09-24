package node

import (
	"crypto/rand"
	"fmt"

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

func ensureWGPrivKey(cn *CentralNetwork) error {
	if cn.myPrivKey != nil {
		return nil
	}
	myWGPrivKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return fmt.Errorf("gen privkey: %w", err)
	}
	cn.myPrivKey = &myWGPrivKey
	return nil
}

type networkPeerPair struct {
	network string
	peer    string
}

func (n *Node) ReplaceCC(cc2 *CentralConfig) {
	n.ccLock.Lock()
	defer n.ccLock.Unlock()
	n.cc = *cc2
}
