package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/nyiyui/qanms/node"
)

var addr string

func main() {
	flag.StringVar(&addr, "addr", ":8080", "bind address")
	flag.Parse()

	s, err := node.New(node.Config{})
	if err != nil {
		panic(err)
	}
	log.Fatal(http.ListenAndServe(addr, s))
}
