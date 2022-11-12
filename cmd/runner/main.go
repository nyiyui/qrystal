package main

import (
	"log"

	"github.com/nyiyui/qrystal/profile"
	"github.com/nyiyui/qrystal/runner"
	"github.com/nyiyui/qrystal/util"
)

func main() {
	log.SetPrefix("main: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	util.ShowCurrent()
	profile.Profile()

	runner.Main()
}
