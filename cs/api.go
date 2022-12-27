package cs

import (
	"errors"
	"fmt"

	"github.com/cenkalti/rpc2"
	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/util"
)

func (c *CentralSource) newHandler() {
	h := rpc2.NewServer()
	h.Handle("ping", c.ping)
	h.Handle("sync", c.sync)
	c.handler = h
}

func (c *CentralSource) ping(cl *rpc2.Client, q *bool, s *bool) error {
	*s = true
	return nil
}

func (c *CentralSource) sync(cl *rpc2.Client, q *api.PullQ, s *api.PullS) error {
	util.S.Infof("sync(%#v)", q)
	//return errors.New("fml")
	util.S.Info("getToken…")
	ti, ok, err := c.Tokens.getToken(q.CentralToken)
	util.S.Info("getToken5", err)
	if err != nil {
		return err
	}
	util.S.Info("getToken6", err)
	if !ok {
		return newTokenAuthError(q.CentralToken)
	}
	util.S.Info("getToken7", err)
	if !ti.CanPull {
		util.S.Info("getToken7a", err)
		return fmt.Errorf("token %s cannot pull", ti.Name)
	}
	util.S.Info("StartUse…")
	ti.StartUse()
	util.S.Info("getToken8", err)
	err = c.Tokens.UpdateToken(ti)
	if err != nil {
		return err
	}
	defer func() {
		ti.StopUse()
		util.S.Infof("UpdateToken1")
		err = c.Tokens.UpdateToken(ti)
		util.S.Infof("UpdateToken2")
		if err != nil {
			util.S.Infof("UpdateToken2a")
			util.S.Errorf("UpdateToken %s: %s", ti.sum, err)
		}
		util.S.Infof("UpdateToken2b")
	}()
	util.S.Infof("%sから新たな認証済プル", ti.Name)

	newCC, err := c.copyCC(ti.Networks)
	if err != nil {
		util.S.Infof("copyCC: %s", err)
		return errors.New("copyCC failed")
	}

	grcns, err := c.generationRequests(ti.Networks)
	if err != nil {
		util.S.Infof("generationRequests (possibly invalid config): %s", err)
		return errors.New("generationRequests failed (possibly invalid config)")
	}
	var s3 api.PushS
	if len(grcns) != 0 {
		util.S.Infof("pull %s: doing priming push for requesting generation of keys", ti.Name)
		err = cl.Call("push", &api.PushQ{I: q.I, CC: *newCC}, &s3)
		if err != nil {
			util.S.Infof("push: %s", err)
			return errors.New("push failed")
		}
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
	err = cl.Call("push", &api.PushQ{CC: *newCC}, &s3)
	if err != nil {
		util.S.Infof("push: %s", err)
		return errors.New("push failed")
	}

	// TODO: how to notify nodes of changes to cc
	return nil
}
