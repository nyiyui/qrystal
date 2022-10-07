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
	var genConfig bool
	flag.BoolVar(&genConfig, "config", false, "不完全設定を作る。")
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
	fmt.Print("[keys]\n")
	fmt.Printf("public-key  = U_%s\n", pubKeyEnc)
	fmt.Printf("private-key = R_%s\n", seedEnc)
	fmt.Printf("token       = %s\n", token)
	fmt.Printf("hash        = %s\n", hashEnc)
	if genConfig {
		fmt.Print("\n\n\n\n")
		fmt.Printf(`## qrystal/node.conf:
# Private key of this Node. Auto-generated using qrystal-gen-keys -config.
#private-key: %s
# Address of Node gRPC server.
#addr: :39251
#cs:
#  host: <qrystal-cs>:39252
#  token: %s
`, pubKeyEnc, hashEnc)
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
