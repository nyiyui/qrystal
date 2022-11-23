package verify

import (
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"github.com/nyiyui/qrystal/node/api"
)

func newXchQ() api.XchQ {
	return api.XchQ{
		PubKey: []byte("pubKey"),
		Psk:    []byte("psk"),
		Cnn:    "cnn",
		Peer:   "peer",
		Ts:     time.Now().Add(-Grave * 2).Format(time.RFC3339),
		Sig:    nil,
	}
}

func TestSignVerifyXchQSig(t *testing.T) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(t)
	}
	t.Run("old", func(t *testing.T) {
		q := newXchQ()
		err = SignXchQ(privKey, &q)
		if err != nil {
			t.Fatal(t)
		}
		err = VerifyXchQ(pubKey, &q)
		if !errors.Is(err, ErrTooFar) {
			t.Fatalf("err must be ErrTooFar: %s", err)
		}
	})
	t.Run("new", func(t *testing.T) {
		q := newXchQ()
		err = SignXchQ(privKey, &q)
		if err != nil {
			t.Fatal(t)
		}
		err = VerifyXchQ(pubKey, &q)
		if !errors.Is(err, ErrTooFar) {
			t.Fatalf("err must be ErrTooFar: %s", err)
		}
	})
	t.Run("sig", func(t *testing.T) {
		q := newXchQ()
		err = SignXchQ(privKey, &q)
		if err != nil {
			t.Fatal(t)
		}
		q.Sig = q.Sig[:len(q.Sig)-2]
		err = VerifyXchQ(pubKey, &q)
		if !errors.Is(err, ErrTooFar) {
			t.Fatalf("err must be ErrTooFar: %s", err)
		}
	})
}

func newXchS() api.XchS {
	return api.XchS{
		PubKey: []byte("pubKey"),
		Ts:     time.Now().Add(-Grave * 2).Format(time.RFC3339),
		Sig:    nil,
	}
}

func TestSignVerifyXchSSig(t *testing.T) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(t)
	}
	t.Run("old", func(t *testing.T) {
		s := newXchS()
		err = SignXchS(privKey, &s)
		if err != nil {
			t.Fatal(t)
		}
		err = VerifyXchS(pubKey, &s)
		if !errors.Is(err, ErrTooFar) {
			t.Fatalf("err must be ErrTooFar: %s", err)
		}
	})
	t.Run("new", func(t *testing.T) {
		s := newXchS()
		err = SignXchS(privKey, &s)
		if err != nil {
			t.Fatal(t)
		}
		err = VerifyXchS(pubKey, &s)
		if !errors.Is(err, ErrTooFar) {
			t.Fatalf("err must be ErrTooFar: %s", err)
		}
	})
	t.Run("sig", func(t *testing.T) {
		s := newXchS()
		err = SignXchS(privKey, &s)
		if err != nil {
			t.Fatal(t)
		}
		s.Sig = s.Sig[:len(s.Sig)-2]
		err = VerifyXchS(pubKey, &s)
		if !errors.Is(err, ErrTooFar) {
			t.Fatalf("err must be ErrTooFar: %s", err)
		}
	})
}
