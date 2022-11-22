package node

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/node/api"
)

type authConn interface {
	Send(*api.AuthSQ) error
	Recv() (*api.AuthSQ, error)
}

type authState struct {
	coordPrivKey ed25519.PrivateKey
	conn         authConn
	cc           central.Config

	// the following are dynamically set by authYours

	cn  *central.Network
	you *central.Peer
}

func (s *authState) solveChall() error {
	sq1Raw, err := s.conn.Recv()
	if err != nil {
		return err
	}
	sq1 := sq1Raw.Sq.(*api.AuthSQ_Q).Q
	var ok bool
	s.cn, ok = s.cc.Networks[sq1.Network]
	if !ok {
		return fmt.Errorf("unknown network: %s", sq1.Network)
	}
	if sq1.You != s.cn.Me {
		return errors.New("not me")
	}
	s.you, ok = s.cn.Peers[sq1.Me]
	if !ok {
		return errors.New("unknown you")
	}

	var added []byte
	var challResp []byte
	{
		added, err = readRand(32)
		if err != nil {
			return errors.New("generating challenge added failed")
		}

		signThis := make([]byte, 64)
		copy(signThis, sq1.Chall)
		copy(signThis[32:], added)
		challResp = ed25519.Sign(s.coordPrivKey, signThis)
	}

	sq2 := api.AuthS{
		ChallResp:  challResp,
		ChallAdded: added,
	}
	err = s.conn.Send(&api.AuthSQ{Sq: &api.AuthSQ_S{S: &sq2}})
	if err != nil {
		return err
	}
	return nil
}

func (s *authState) verifyChall(cnn, yourName string) error {
	if s.you == nil {
		return errors.New("authMine: authState.you is nil")
	}
	chall, err := readRand(32)
	if err != nil {
		return errors.New("gen chall failed")
	}
	sq3 := api.AuthQ{
		Network: cnn,
		Me:      s.cn.Me,
		You:     yourName,
		Chall:   chall,
	}
	err = s.conn.Send(&api.AuthSQ{Sq: &api.AuthSQ_Q{Q: &sq3}})
	if err != nil {
		return err
	}
	sq4Raw, err := s.conn.Recv()
	if err != nil {
		return err
	}
	sq4 := sq4Raw.Sq.(*api.AuthSQ_S).S
	signed := make([]byte, 64)
	copy(signed, chall)
	copy(signed[32:], sq4.ChallAdded)
	ok := ed25519.Verify(ed25519.PublicKey(s.you.PublicKey), signed, sq4.ChallResp)
	if !ok {
		return errors.New("signature verification failed")
	}
	return nil
}
