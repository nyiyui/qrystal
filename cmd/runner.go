package main

import (
	"log"

	"github.com/nyiyui/qanms/runner"
)

func main() {
	log.SetPrefix("main: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	runner.Main()
}
