package cs

import (
	"io"
	"sync"

	"github.com/cenkalti/rpc2"

	"github.com/nyiyui/qrystal/central"
	"github.com/tidwall/buntdb"
)

type CentralSource struct {
	cc           central.Config
	ccLock       sync.RWMutex
	Tokens       TokenStore
	backportLock sync.Mutex
	// keep it simple (RWMutex might be more appropriate but we're not writing
	// backports simultaneously (I hope))
	backportPath  string
	handler       *rpc2.Server
	notifyChsLock sync.Mutex
	notifyChs     []chan change
}

func New(cc central.Config, backportPath string, db *buntdb.DB) (*CentralSource, error) {
	cs := new(CentralSource)
	ts, err := newTokenStore(db, cs)
	if err != nil {
		return nil, err
	}
	cs.Tokens = ts

	for _, cn := range cc.Networks {
		for _, peer := range cn.Peers {
			peer.Internal = new(central.PeerInternal)
		}
	}

	cs.cc = cc
	cs.backportPath = backportPath
	cs.newHandler()
	return cs, nil
}

func (s *CentralSource) serve(conn io.ReadWriteCloser) {
	s.handler.ServeConn(conn)
}

func SliceToMap(ss []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}

func MissingFromFirst[T any](m1 map[string]T, m2 map[string]T) []string {
	r := []string{}
	for k := range m2 {
		if _, ok := m1[k]; !ok {
			r = append(r, k)
		}
	}
	return r
}

func contains(ss []string, s string) bool {
	for _, s2 := range ss {
		if s == s2 {
			return true
		}
	}
	return false
}
