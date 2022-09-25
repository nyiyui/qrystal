package main

import (
	"log"

	"github.com/nyiyui/qrystal/mio"
)

func main() {
	log.SetPrefix("mio:  ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	err := mio.Main()
	if err != nil {
		log.Fatalf("%s", err)
	}
}
