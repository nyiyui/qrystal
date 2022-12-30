package main

import (
	"fmt"

	"github.com/nyiyui/qrystal/util"
)

func main() {
	fmt.Println("sd-notify-test")
	fmt.Println("notifying")
	err := util.Notify("READY=1\nSTATUS=status")
	if err != nil {
		panic(err)
	}
	fmt.Println("notified")
	select {}
}
