package main

import (
	"log"
	"os"

	"golang.zx2c4.com/wireguard/wgctrl"
)

func main() {
	client, err := wgctrl.New()
	if err != nil {
		panic(err)
	}
	defer client.Close()
	dev, err := client.Device(os.Args[1])
	if err != nil {
		panic(err)
	}
	log.Println("dev", dev)
}
