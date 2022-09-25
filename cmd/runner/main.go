package main

import (
	"log"

	"github.com/nyiyui/qrystal/runner"
)

func main() {
	log.SetPrefix("main: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	runner.Main()
}
