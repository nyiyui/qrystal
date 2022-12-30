package node

import "testing"

func TestCommands(t *testing.T) {
	if CommandIp == "" {
		t.Fatal("CommandIp is blank")
	}
	if CommandIptables == "" {
		t.Fatal("CommandIptables is blank")
	}
}
