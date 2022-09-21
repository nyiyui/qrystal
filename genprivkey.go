package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log"
)

func main() {
	pubkey, privkey, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatalf("generating key: %s", err)
		return
	}
	pubkey2 := make([]byte, base64.StdEncoding.EncodedLen(len(pubkey)))
	privkey2 := make([]byte, base64.StdEncoding.EncodedLen(len(privkey)))
	base64.StdEncoding.Encode(pubkey2, pubkey)
	base64.StdEncoding.Encode(privkey2, privkey.Seed())
	fmt.Printf("public key: U_%s\n", pubkey2)
	fmt.Printf("private key (seed): R_%s\n", privkey2)
}
