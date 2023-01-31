package util

import (
	"bytes"
	"testing"
)

func TestToken(t *testing.T) {
	tok, err := RandomToken()
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := ParseToken(tok.String())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tok.raw, tok2.raw) {
		t.Fatalf("mismatch:\ntok: %s\ntok2: %s", tok, tok2)
	}
}
func TestTokenHash(t *testing.T) {
	tok, err := RandomToken()
	if err != nil {
		t.Fatal(err)
	}
	th := tok.Hash()
	th2, err := ParseTokenHash(th.String())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(th.raw[:], th2.raw[:]) {
		t.Fatalf("mismatch:\nth: %s\nth2: %s", th, th2)
	}
}
