package cs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/cenkalti/rpc2"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"github.com/tidwall/buntdb"
)

type CentralSource struct {
	api.UnimplementedCentralSourceServer
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

var _ api.CentralSourceServer = new(CentralSource)

func (s *CentralSource) Ping(ctx context.Context, ss *api.PingQS) (*api.PingQS, error) {
	return &api.PingQS{}, nil
}

func FromIPNets(nets []net.IPNet) (dest []*api.IPNet) {
	dest = make([]*api.IPNet, len(nets))
	for i, n := range nets {
		if n.String() == "<nil>" {
			panic("nil IPNet")
		}
		dest[i] = &api.IPNet{Cidr: n.String()}
	}
	return
}

func ToIPNets(nets []*api.IPNet) (dest []net.IPNet, err error) {
	dest = make([]net.IPNet, len(nets))
	for i, n := range nets {
		dest[i], err = util.ParseCIDR(n.Cidr)
		if err != nil {
			return nil, err
		}
	}
	return
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

func (s *CentralSource) Push(ctx context.Context, q *api.PushQ) (*api.PushS, error) {
	ti, ok, err := s.Tokens.getToken(q.CentralToken)
	if err != nil {
		return &api.PushS{S: &api.PushS_Other{Other: fmt.Sprint(err)}}, err
	}
	if !ok {
		return nil, newTokenAuthError(q.CentralToken)
	}
	if ti.CanPush == nil {
		return nil, errors.New("cannot push")
	}
	if !ti.CanPush.Any {
		cpn, ok := ti.CanPush.Networks[q.Cnn]
		if !ok {
			return nil, fmt.Errorf("cannot push to net %s", q.Cnn)
		}
		if q.PeerName != cpn.Name {
			return nil, fmt.Errorf("cannot push to net %s peer %s", q.Cnn, q.PeerName)
		}
		if cpn.CanSeeElement != nil {
			if q.Peer.CanSee == nil {
				return nil, fmt.Errorf("cannot push to net %s as peer violates CanSeeElement any", q.Cnn)
			} else if len(MissingFromFirst(SliceToMap(cpn.CanSeeElement), SliceToMap(q.Peer.CanSee.Only))) != 0 {
				return nil, fmt.Errorf("cannot push to net %s as peer violates CanSeeElement %s", q.Cnn, cpn.CanSeeElement)
			}
		}
	}

	if ti.Networks == nil {
		ti.Networks = map[string]string{}
	}
	ti.Networks[q.Cnn] = q.PeerName

	ti.Use()
	err = s.Tokens.UpdateToken(ti)
	if err != nil {
		return nil, err
	}

	util.S.Infof("push %#v", q)

	peer, err := central.NewPeerFromAPI(q.PeerName, q.Peer)
	if err != nil {
		return &api.PushS{
			S: &api.PushS_InvalidData{
				InvalidData: fmt.Sprint(err),
			},
		}, nil
	}
	if _, ok := s.cc.Networks[q.Cnn]; !ok {
		return nil, fmt.Errorf("unknown net %s", q.Cnn)
	}
	pushS, err := func() (*api.PushS, error) {
		s.ccLock.Lock()
		defer s.ccLock.Unlock()
		cn := s.cc.Networks[q.Cnn]
		// TODO: impl checks for PublicKey, host, net overlap
		cn.Peers[q.PeerName] = peer
		return nil, nil
	}()
	if err != nil {
		return nil, err
	}
	if pushS != nil {
		return pushS, nil
	}
	util.S.Infof("push net %s peer %s: notify change", q.Cnn, q.PeerName)
	return &api.PushS{
		S: &api.PushS_Ok{},
	}, nil
}

func (s *CentralSource) AddToken(ctx context.Context, q *api.AddTokenQ) (*api.AddTokenS, error) {
	ti, ok, err := s.Tokens.getToken(q.CentralToken)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, newTokenAuthError(q.CentralToken)
	}
	ti.Use()
	err = s.Tokens.UpdateToken(ti)
	if err != nil {
		return nil, err
	}
	if ti.CanAddTokens == nil {
		return nil, errors.New("cannot add tokens")
	}
	var hash sha256Sum
	n := copy(hash[:], q.Hash)
	if n != len(hash) {
		return nil, fmt.Errorf("hash %d length invalid (expected %d)", n, len(hash))
	}
	util.S.Infof("add token %s: %s", q.Name, q)
	err = s.Tokens.AddToken(hash, TokenInfo{
		Name:     q.Name,
		Networks: q.Networks,
		CanPull:  q.CanPull,
		CanPush:  convCanPush(q.CanPush, q.CanPushNetworksCanSeeElement),
	}, q.Overwrite)
	if err != nil {
		return nil, err
	}
	return &api.AddTokenS{}, nil
}

func convCanPush(c *api.CanPush, ncse map[string]*api.LString) *CanPush {
	if c == nil {
		return nil
	}
	networks := map[string]CanPushNetwork{}
	for key, name := range c.Networks {
		cpn := CanPushNetwork{Name: name}
		if cse := ncse[key]; cse != nil {
			cpn.CanSeeElement = cse.Inner
		}
		networks[key] = cpn
	}
	return &CanPush{
		Networks: networks,
	}
}

func contains(ss []string, s string) bool {
	for _, s2 := range ss {
		if s == s2 {
			return true
		}
	}
	return false
}
