package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
)

func main() {
	var formatJson bool
	flag.BoolVar(&formatJson, "json", false, "JSONを出力します。")
	flag.Parse()

	token, hash, err := genHash()
	if err != nil {
		log.Fatalf("gen hash/token: %s", err)
	}
	hashEnc := hex.EncodeToString(hash[:])
	if formatJson {
		fmt.Printf(`{
  "keys": {
		"token": "%s",
		"hash": "%s"
	}
}`, token, hashEnc)
	} else {
		fmt.Print("[keys]\n")
		fmt.Printf("token = %s\n", token)
		fmt.Printf("hash  = %s\n", hashEnc)
	}
}

func genHash() (token string, hash [sha256.Size]byte, err error) {
	token2 := make([]byte, 64)
	_, err = rand.Read(token2)
	if err != nil {
		return
	}
	token = base64.StdEncoding.EncodeToString(token2)
	hash = sha256.Sum256([]byte(token))
	return
}
