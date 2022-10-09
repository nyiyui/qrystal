package util

import (
	"errors"
	"math/big"
	"net"
)

var AddressOverflow = errors.New("overflowed network")

func pow(a, b int) int {
	if a == 0 {
		return 0
	}
	r := a
	for i := 2; i <= b; i++ {
		r *= a
	}
	return r
}

func AssignAddress(ipNet *net.IPNet, usedIPs []net.IPNet) (ip net.IP, err error) {
	// TODO: performance improvements?
	cand := ipNet.IP
NextIP:
	for {
		if !ipNet.Contains(cand) {
			return nil, AddressOverflow
		}
		for _, usedIP := range usedIPs {
			if usedIP.Contains(cand) {
				cand = nextIP(cand)
				continue NextIP
			}
		}
		// not used
		return cand, nil
	}
}

func nextIP(ip net.IP) net.IP {
	n := big.NewInt(0).SetBytes([]byte(ip))
	n.Add(n, big.NewInt(1))
	b := n.Bytes()
	// add leading zeroes
	b = append(make([]byte, len(ip)-len(b)), b...)
	return b
}
