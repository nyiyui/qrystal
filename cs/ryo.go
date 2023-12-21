package cs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nyiyui/qrystal/api"
	"github.com/nyiyui/qrystal/util"
)

type tokenInfoKeyType struct{}

var tokenInfoKey = tokenInfoKeyType{}

func (c *CentralSource) HandleRyo(addr string, tlsCfg TLS) error {
	util.S.Info("ryo: starting…")
	if addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	// TODO: only POST
	mux.Handle("/push", c.ryoToken(http.HandlerFunc(c.ryoPush)))
	mux.Handle("/add-token", c.ryoToken(http.HandlerFunc(c.ryoAddToken)))
	mux.Handle("/remove-token", c.ryoToken(http.HandlerFunc(c.ryoRemoveToken)))
	s := &http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: 1 * time.Second,
		ErrorLog:    log.New(os.Stderr, "ryo server: ", log.Lmsgprefix|log.LstdFlags|log.Lshortfile),
	}
	go func() {
		err := s.ListenAndServeTLS(tlsCfg.CertPath, tlsCfg.KeyPath)
		if err != nil {
			util.S.Fatalf("ryo: serve failed: %s", err)
		}
	}()
	util.S.Info("ryo: serving")
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
	if r.Method != "POST" {
		http.Error(w, "only POST", 405)
		return
	}
	var q api.HPushQ
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		util.S.Errorf("json decode failed: %s", err)
		http.Error(w, "json decode failed", 422)
		return
	}

	ti := r.Context().Value(tokenInfoKey).(TokenInfo)
	if ti.CanPush == nil {
		http.Error(w, "token: cannot push", 403)
		return
	}
	q.Peer.Name = q.PeerName
	c.ccLock.RLock()
	defer c.ccLock.RUnlock()
	err = checkPeer(ti, q.CNN, c.cc, q.Peer)
	if err != nil {
		switch err := err.(type) {
		case httpError:
			http.Error(w, fmt.Sprint(err.err), err.code)
		default:
			http.Error(w, fmt.Sprint(err), 500)
		}
		return
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
}

func (c *CentralSource) ryoAddToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST", 405)
		return
	}
	var q api.HAddTokenQ
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		util.S.Errorf("json decode failed: %s", err)
		http.Error(w, "json decode failed", 422)
		return
	}

	ti := r.Context().Value(tokenInfoKey).(TokenInfo)
	if ti.CanAdminTokens == nil {
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
	canPush := new(CanPush)
	if len(q.CanPush) != 0 {
		canPush.Networks = map[string]CanPushNetwork{}
		for cnn, can := range q.CanPush {
			canPush.Networks[cnn] = CanPushNetwork{
				Name:          can.PeerName,
				CanSeeElement: can.CanSeeElementOf,
			}
		}
	}
	err = c.Tokens.AddToken(*q.Hash, TokenInfo{
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
	w.Write([]byte("ok"))
}

func (c *CentralSource) ryoRemoveToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST", 405)
		return
	}
	var q api.HRemoveTokenQ
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		util.S.Errorf("json decode failed: %s", err)
		http.Error(w, "json decode failed", 422)
		return
	}

	ti := r.Context().Value(tokenInfoKey).(TokenInfo)
	if ti.CanAdminTokens == nil {
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
	util.S.Infof("remove token %s: %s", q.Hash, q)
	err = c.Tokens.RemoveToken(*q.Hash)
	if err != nil {
		http.Error(w, "add token failed", 500)
		util.S.Errorf("RemoveToken: %s", err)
		return
	}
	w.Write([]byte("ok"))
}
