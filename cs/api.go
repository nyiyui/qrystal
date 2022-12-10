package cs

import (
	"errors"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/util"
)

func (c *CentralSource) newHandler() {
	h := rpc2.NewServer()
	c.handler = h
	h.Handle("pull", c.pull)
}

func (c *CentralSource) pull(cl *rpc2.Client, q *api.PullQ, s *api.PullS) error {
	ti, ok, err := c.Tokens.getToken(q.CentralToken)
	if err != nil {
		return err
	}
	if !ok {
		return newTokenAuthError(q.CentralToken)
	}
	if !ti.CanPull {
		return errors.New("cannot pull")
	}
	ti.StartUse()
	err = c.Tokens.UpdateToken(ti)
	if err != nil {
		return err
	}
	defer func() {
		ti.StopUse()
		err = c.Tokens.UpdateToken(ti)
		if err != nil {
			util.S.Errorf("UpdateToken %s: %s", ti.sum, err)
		}
	}()
	util.S.Infof("%sから新たな認証済プル", ti.Name)
	// token status could change while this is called
	ti, ok, err = c.Tokens.getToken(q.CentralToken)
	if err != nil {
		return err
	}
	if !ok {
		return newTokenAuthError(q.CentralToken)
	}
	if !ti.CanPull {
		return errors.New("cannot pull")
	}
	util.S.Infof("%sに送ります。", ti.Name)

	newCC, err := c.copyCC(ti.Networks)
	if err != nil {
		util.S.Infof("convertCC: %s", err)
		return errors.New("conversion failed")
	}
	s.CC = *newCC
	return nil
}
