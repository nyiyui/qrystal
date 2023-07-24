package node

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

// Note: ccLock must be held.
func (n *Node) forwardingRequired(cnn string) bool {
	// TODO: make this so it only returns true when there is a node that is not
	//       accessible from n > 0 nodes, and there is a path to those nodes.
	//       This requires more communication between nodes to agree on a path
	//       between those nodes.
	cn, ok := n.cc.Networks[cnn]
	if !ok {
		panic("cnn invalid")
	}
	for _, peer := range cn.Peers {
		if peer.Host == "" {
			return true
		}
	}
	return false
}

// nominateForwarder returns an index of a peer that is apparently the best for forwarding WireGuard connections.
// Currently, the function chooses a peer that can forward by random.
// Note: ccLock must be held.
// TODO: don't select peers that handshaked more than Keepalive seconds ago (consider moving this nominating process to mio)
func (n *Node) nominateForwarder(cnn string) (peerName string, err error) {
	cn, ok := n.cc.Networks[cnn]
	if !ok {
		panic("cnn invalid")
	}
	available := make([]string, 0, len(cn.Peers))
	for pn, peer := range cn.Peers {
		if peer.CanForward {
			available = append(available, pn)
		}
	}
	if len(available) == 0 {
		return "", errors.New("no available forwarders")
	}
	i, err := rand.Int(rand.Reader, big.NewInt(int64(len(available)-1)))
	if err != nil {
		return "", fmt.Errorf("choosing random peer: %w", err)
	}
	return available[int(i.Int64())], nil
}
