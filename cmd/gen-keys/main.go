package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/nyiyui/qrystal/util"
)

func main() {
	var formatJson bool
	flag.BoolVar(&formatJson, "json", false, "JSONを出力します。")
	flag.Parse()

	token, err := util.RandomToken()
	if err != nil {
		log.Fatalf("gen hash/token: %s", err)
	}
	if formatJson {
		fmt.Printf(`{
  "keys": {
		"token": "%s",
		"hash": "%s"
	}
}`, token, token.Hash().String())
	} else {
		fmt.Print("[keys]\n")
		fmt.Printf("token = %s\n", token)
		fmt.Printf("hash  = %s\n", token.Hash().String())
	}
}
