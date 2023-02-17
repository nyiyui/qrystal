package main

import (
	"log"

	"github.com/nyiyui/qrystal/hokuto"
	"github.com/nyiyui/qrystal/profile"
	"github.com/nyiyui/qrystal/util"
	"go.uber.org/zap/zapcore"
)

func main() {
	util.SetupLog()
	defer util.S.Sync()
	util.Atom.SetLevel(zapcore.InfoLevel)
	log.SetPrefix("hokuto:  ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	util.ShowCurrent()
	profile.Profile()

	err := hokuto.Main()
	if err != nil {
		log.Fatalf("%s", err)
	}
}
