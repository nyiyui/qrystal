package main

import "github.com/nyiyui/qrystal/util"

func main() {
	err := util.Notify("READY=1\nSTATUS=status")
	if err != nil {
		panic(err)
	}
}
