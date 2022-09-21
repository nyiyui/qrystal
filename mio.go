package main

import (
	"log"

	"github.com/nyiyui/qanms/mio"
)

func main() {
	err := mio.Main()
	if err != nil {
		log.Fatalf("%s", err)
	}
}
