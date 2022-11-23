package verify

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nyiyui/qrystal/node/api"
)

const Grave = 5 * time.Second

var (
	ErrTooFar     = errors.New("too far")
	ErrInvalidSig = errors.New("invalid sig")
)

type xchQSigPayload struct {
	PubKey []byte `json:"pubKey"`
	Psk    []byte `json:"psk"`
	Cnn    string `json:"cnn"`
	Peer   string `json:"peer"`
	Ts     string `json:"ts"`
}

func newXchQPayload(q *api.XchQ) xchQSigPayload {
	return xchQSigPayload{
		PubKey: q.PubKey,
		Psk:    q.Psk,
		Cnn:    q.Cnn,
		Peer:   q.Peer,
		Ts:     q.Ts,
	}
}

func VerifyXchQ(signedBy ed25519.PublicKey, q *api.XchQ) (err error) {
	err = verifyTs(q.Ts)
	if err != nil {
		return err
	}
	payload2, err := json.Marshal(newXchQPayload(q))
	if err != nil {
		return err
	}
	ok := ed25519.Verify(signedBy, payload2, q.Sig)
	if !ok {
		return ErrInvalidSig
	}
	return nil
}

func SignXchQ(signUsing ed25519.PrivateKey, q *api.XchQ) (err error) {
	payload2, err := json.Marshal(newXchQPayload(q))
	if err != nil {
		return err
	}
	q.Sig = ed25519.Sign(signUsing, payload2)
	return nil
}

type xchSSigPayload struct {
	PubKey []byte `json:"pubKey"`
	Ts     string `json:"ts"`
}

func newXchSPayload(s *api.XchS) xchSSigPayload {
	return xchSSigPayload{
		PubKey: s.PubKey,
		Ts:     s.Ts,
	}
}

func VerifyXchS(signedBy ed25519.PublicKey, s *api.XchS) (err error) {
	err = verifyTs(s.Ts)
	if err != nil {
		return err
	}
	payload2, err := json.Marshal(newXchSPayload(s))
	if err != nil {
		return err
	}
	ok := ed25519.Verify(signedBy, payload2, s.Sig)
	if !ok {
		return ErrInvalidSig
	}
	return nil
}

func SignXchS(signUsing ed25519.PrivateKey, s *api.XchS) (err error) {
	payload2, err := json.Marshal(newXchSPayload(s))
	if err != nil {
		return err
	}
	s.Sig = ed25519.Sign(signUsing, payload2)
	return nil
}

func verifyTs(ts string) (err error) {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	delay := time.Since(t)
	if delay >= Grave || delay < 0 {
		return fmt.Errorf("%w: %s", ErrTooFar, delay)
	}
	return nil
}
