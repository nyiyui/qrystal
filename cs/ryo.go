package cs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type tokenInfoKeyType struct{}

var tokenInfoKey = tokenInfoKeyType{}

func (c *CentralSource) HandleRyo(addr string, tlsCfg TLS) error {
	if addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	// TODO: only POST
	mux.Handle("/push", c.ryoToken(http.HandlerFunc(c.ryoPush)))
	mux.Handle("/add-token", c.ryoToken(http.HandlerFunc(c.ryoAddToken)))
	el, err := zap.NewStdLogAt(util.L, zapcore.ErrorLevel)
	if err != nil {
		panic(err)
	}
	s := &http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: 1 * time.Second,
		ErrorLog:    el,
	}
	go func() {
		err := s.ListenAndServeTLS(tlsCfg.CertPath, tlsCfg.KeyPath)
		if err != nil {
			util.S.Fatalf("ryo: serve failed: %s", err)
		}
	}()
	util.S.Info("ryo: serving…")
	return nil
}

func (c *CentralSource) ryoToken(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctRaw := r.Header.Get("X-Qrystal-CentralToken")
		ct, err := util.ParseToken(ctRaw)
		if err != nil {
			http.Error(w, "parse token failed", 422)
			return
		}
		ti, ok, err := c.Tokens.getToken(ct)
		if err != nil {
			http.Error(w, "get token failed", 401)
			return
		}
		if !ok {
			http.Error(w, "token not found", 401)
			return
		}
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tokenInfoKey, ti)))
	})
}

func (c *CentralSource) ryoPush(w http.ResponseWriter, r *http.Request) {
	var q api.HPushQ
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, "json decode failed", 422)
		return
	}

	ti := r.Context().Value(tokenInfoKey).(TokenInfo)
	if ti.CanPush == nil {
		http.Error(w, "token: cannot push", 403)
		return
	}
	if !ti.CanPush.Any {
		cpn, ok := ti.CanPush.Networks[q.CNN]
		if !ok {
			http.Error(w, fmt.Sprintf("cannot push to net %s", q.CNN), 403)
			return
		}
		if q.PeerName != cpn.Name {
			http.Error(w, fmt.Sprintf("cannot push to net %s peer %s", q.CNN, q.PeerName), 403)
			return
		}
		if cpn.CanSeeElement != nil {
			if q.Peer.CanSee == nil {
				http.Error(w, fmt.Sprintf("cannot push to net %s as peer violates CanSeeElement any", q.CNN), 403)
				return
			} else if len(MissingFromFirst(SliceToMap(cpn.CanSeeElement), SliceToMap(q.Peer.CanSee.Only))) != 0 {
				http.Error(w, fmt.Sprintf("cannot push to net %s as peer violates CanSeeElement %s", q.CNN, cpn.CanSeeElement), 403)
				return
			}
		}
	}

	if ti.Networks == nil {
		ti.Networks = map[string]string{}
	}
	ti.Networks[q.CNN] = q.PeerName

	ti.Use()
	err = c.Tokens.UpdateToken(ti)
	if err != nil {
		util.S.Errorf("UpdateToken: %s", err)
		http.Error(w, "update token failed", 500)
		return
	}

	util.S.Infof("push %#v", q)

	if _, ok := c.cc.Networks[q.CNN]; !ok {
		http.Error(w, fmt.Sprintf("unknown net %s", q.CNN), 422)
		return
	}
	func() {
		c.ccLock.Lock()
		defer c.ccLock.Unlock()
		cn := c.cc.Networks[q.CNN]
		// TODO: impl checks for PublicKey, host, net overlap
		cn.Peers[q.PeerName] = &q.Peer
	}()
	util.S.Infof("push net %s peer %s: notify change", q.CNN, q.PeerName)
	w.Write([]byte("ok"))
	return
}

func (c *CentralSource) ryoAddToken(w http.ResponseWriter, r *http.Request) {
	var q api.HAddTokenQ
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, "json decode failed", 422)
		return
	}

	ti := r.Context().Value(tokenInfoKey).(TokenInfo)
	if ti.CanAddTokens == nil {
		http.Error(w, "token: cannot push", 403)
		return
	}
	ti.Use()
	err = c.Tokens.UpdateToken(ti)
	if err != nil {
		util.S.Errorf("UpdateToken: %s", err)
		http.Error(w, "update token failed", 500)
		return
	}
	util.S.Infof("add token %s: %s", q.Name, q)
	var canPush *CanPush
	if len(q.CanPush) != 0 {
		canPush.Networks = map[string]CanPushNetwork{}
		for cnn, can := range q.CanPush {
			canPush.Networks[cnn] = CanPushNetwork{
				Name:          can.PeerName,
				CanSeeElement: can.CanSeeElementOf,
			}
		}
	}
	err = c.Tokens.AddToken(q.Hash, TokenInfo{
		Name:     q.Name,
		Networks: q.CanPull,
		CanPull:  len(q.CanPull) != 0,
		CanPush:  canPush,
	}, q.Overwrite)
	if err != nil {
		http.Error(w, "add token failed", 500)
		util.S.Errorf("AddToken: %s", err)
		return
	}
	return
}
