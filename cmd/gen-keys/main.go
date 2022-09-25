package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
)

func main() {
	token, hash, err := genHash()
	if err != nil {
		log.Fatalf("gen hash/token: %s", err)
	}
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("gen key pair: %s", err)
	}
	fmt.Print("[keys]\n")
	fmt.Printf("public-key  = U_%s\n", base64.StdEncoding.EncodeToString(pubKey))
	fmt.Printf("private-key = R_%s\n", base64.StdEncoding.EncodeToString(privKey.Seed()))
	fmt.Printf("token       = %s\n", token)
	fmt.Printf("hash        = %s\n", hex.EncodeToString(hash[:]))
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
