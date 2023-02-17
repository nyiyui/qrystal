package main

import (
	"log"

	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/profile"
	"github.com/nyiyui/qrystal/util"
)

func main() {
	util.SetupLog()
	defer util.S.Sync()
	log.SetPrefix("mio:  ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	util.ShowCurrent()
	profile.Profile()

	err := mio.Main()
	if err != nil {
		log.Fatalf("%s", err)
	}
}
