package main

import (
	"log"

	"github.com/nyiyui/qanms/mio"
)

func main() {
	log.SetPrefix("mio:  ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	err := mio.Main()
	if err != nil {
		log.Fatalf("%s", err)
	}
}
