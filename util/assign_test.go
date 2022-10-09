package util

import (
	"errors"
	"net"
	"testing"
)

func mustCIDR(s string) net.IPNet {
	_, cidr, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return *cidr
}

func TestAssignAddress(t *testing.T) {
	t.Run("10.2.0.0/16", func(t *testing.T) {
		cidr := mustCIDR("10.2.0.0/16")
		used := []net.IPNet{
			mustCIDR("10.2.0.0/32"),
			mustCIDR("10.2.0.1/32"),
			mustCIDR("10.2.0.2/32"),
		}
		ip, err := AssignAddress(&cidr, used)
		if err != nil {
			t.Fatal(err)
		}
		if !ip.Equal(net.IPv4(10, 2, 0, 3)) {
			t.Fatalf("unexpected IP %s, wanted 10.2.0.3", ip)
		}
	})
	t.Run("check-overflow", func(t *testing.T) {
		cidr := mustCIDR("10.2.0.0/31")
		used := []net.IPNet{
			mustCIDR("10.2.0.0/32"),
			mustCIDR("10.2.0.1/32"),
			mustCIDR("10.2.0.2/32"),
			mustCIDR("10.2.0.3/32"),
		}
		_, err := AssignAddress(&cidr, used)
		if !errors.Is(err, AddressOverflow) {
			t.Fatalf("unexpected error %s, wanted %s", err, AddressOverflow)
		}
	})
}
