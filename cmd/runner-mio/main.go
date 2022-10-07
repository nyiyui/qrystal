package main

import (
	"log"

	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/util"
)

func main() {
	log.SetPrefix("mio:  ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	util.ShowCurrent()

	err := mio.Main()
	if err != nil {
		log.Fatalf("%s", err)
	}
}
