package node

import (
	"sort"
)

func (n *Node) forwardingRequired(cnn string) bool {
	cn := n.cc.Networks[cnn]
	for _, peer := range cn.Peers {
		if peer.Host == "" {
			return true
		}
	}
	return false
}

func (n *Node) canForwardNodes(cnn string) (res []string) {
	res = make([]string, 0)
	cn := n.cc.Networks[cnn]
	for pn, peer := range cn.Peers {
		if peer.CanForward {
			res = append(res, pn)
		}
	}
	sort.Strings(res)
	return
}
