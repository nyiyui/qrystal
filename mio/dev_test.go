package mio

import "testing"

func TestCommands(t *testing.T) {
	if CommandWg == "" {
		t.Fatal("CommandWg is blank")
	}
	if CommandWgQuick == "" {
		t.Fatal("CommandWgQuick is blank")
	}
	if CommandBash == "" {
		t.Fatal("CommandBash is blank")
	}
}
