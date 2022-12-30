package node

import (
	"crypto/tls"
	"errors"
	"regexp"
)

type CSConfig struct {
	Comment         string
	TLSConfig       *tls.Config
	Host            string
	Token           string
	NetworksAllowed []*regexp.Regexp
}

func (csc *CSConfig) netAllowed(cnn string) bool {
	if len(csc.NetworksAllowed) == 0 {
		return true
	}
	for _, pattern := range csc.NetworksAllowed {
		if pattern.MatchString(cnn) {
			return true
		}
	}
	return false
}

func (n *Node) getCSForNet(cnn string) (i int, err error) {
	i, ok := n.csNets[cnn]
	if !ok {
		return 0, errors.New("not found")
	}
	return
}
