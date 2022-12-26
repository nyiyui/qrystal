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

	grcns, err := c.generationRequests(ti.Networks)
	if err != nil {
		util.S.Infof("generationRequests (possibly invalid config): %s", err)
		return errors.New("generationRequests failed (possibly invalid config)")
	}
	if len(grcns) != 0 {
		util.S.Infof("pull %s: requesting generation of keys", ti.Name)
		var s2 api.GenerateS
		err = cl.Call("generate", &api.GenerateQ{CNNs: grcns}, &s2)
		if err != nil {
			util.S.Infof("generate with %s: %s", grcns, err)
			return errors.New("generate failed")
		}
		func() {
			c.ccLock.Lock()
			defer c.ccLock.Unlock()
			for i, key := range s2.PubKeys {
				cnn := grcns[i]
				cn := c.cc.Networks[cnn]
				peer := cn.Peers[ti.Networks[cnn]]
				peer.Internal.PubKey = &key
			}
		}()
	}

	newCC, err := c.copyCC(ti.Networks)
	if err != nil {
		util.S.Infof("convertCC: %s", err)
		return errors.New("conversion failed")
	}
	s.CC = *newCC
	// TODO: how to notify nodes of changes to cc
	return nil
}
