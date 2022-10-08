package node

import (
	"sort"
)

func (n *Node) forwardingRequired(cnn string) bool {
	// TODO: make this so it only returns true when there is a node that is not
	//       accessible from n > 0 nodes, and there is a path to those nodes.
	//       This requires more communication between nodes to agree on a path
	//       between those nodes.
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
