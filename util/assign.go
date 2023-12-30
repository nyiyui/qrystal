package util

import (
	"errors"
	"math/big"
	"net"
	"sort"
)

func ParseCIDR(s string) (net.IPNet, error) {
	ip, cidr, err := net.ParseCIDR(s)
	if err != nil {
		return net.IPNet{}, err
	}
	cidr.IP = ip
	return *cidr, nil
}

var ErrAddressOverflow = errors.New("overflowed network")

func AssignAddress(ipNet *net.IPNet, usedIPs []net.IPNet) (ip net.IP, err error) {
	// TODO: performance improvements?
	cand := ipNet.IP
	sort.Slice(usedIPs)
NextIP:
	for {
		if !ipNet.Contains(cand) {
			return nil, ErrAddressOverflow
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
