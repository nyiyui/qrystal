//go:build !nix

package node

import "testing"

func TestGetOutbound(t *testing.T) {
	outbound, err := getOutbound()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(outbound)
}
