package main

import (
	"crypto/ed25519"
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
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("gen key pair: %s", err)
	}
	pubKeyEnc := base64.StdEncoding.EncodeToString(pubKey)
	seedEnc := base64.StdEncoding.EncodeToString(privKey.Seed())
	hashEnc := hex.EncodeToString(hash[:])
	if formatJson {
		fmt.Printf(`{
  "keys": {
		"public-key": "U_%s",
		"private-key": "R_%s",
		"token": "%s",
		"hash": "%s"
	}
}`, pubKeyEnc, seedEnc, token, hashEnc)
	} else {
		fmt.Print("[keys]\n")
		fmt.Printf("public-key  = U_%s\n", pubKeyEnc)
		fmt.Printf("private-key = R_%s\n", seedEnc)
		fmt.Printf("token       = %s\n", token)
		fmt.Printf("hash        = %s\n", hashEnc)
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
