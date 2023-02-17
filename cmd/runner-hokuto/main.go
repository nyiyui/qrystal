package main

import (
	"log"

	"github.com/nyiyui/qrystal/hokuto"
	"github.com/nyiyui/qrystal/profile"
	"github.com/nyiyui/qrystal/util"
)

func main() {
	util.SetupLog()
	log.SetPrefix("hokuto:  ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	util.ShowCurrent()
	profile.Profile()

	err := hokuto.Main()
	if err != nil {
		log.Fatalf("%s", err)
	}
}
