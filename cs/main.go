package cs

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/rpc2"
	"net"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"github.com/tidwall/buntdb"
)

const commitDelay = 1 * time.Second

type CentralSource struct {
	api.UnimplementedCentralSourceServer
	cc           central.Config
	ccLock       sync.RWMutex
	Tokens       TokenStore
	backportLock sync.Mutex
	// keep it simple (RWMutex might be more appropriate but we're not writing
	// backports simultaneously (I hope))
	backportPath string
	handler      *rpc2.Server
}

func New(cc central.Config, backportPath string, db *buntdb.DB) (*CentralSource, error) {
	ts, err := newTokenStore(db)
	if err != nil {
		return nil, err
	}
	cs := &CentralSource{
		cc:           cc,
		Tokens:       ts,
		backportPath: backportPath,
	}
	cs.newHandler()
	return cs, nil
}

type change struct {
	reason string
	except string
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
		pattern, ok := ti.CanPush.Networks[q.Cnn]
		if !ok {
			return nil, fmt.Errorf("cannot push to net %s", q.Cnn)
		}
		if q.PeerName != pattern {
			return nil, fmt.Errorf("cannot push to net %s peer %s", q.Cnn, q.PeerName)
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
		if len(peer.AllowedIPs) == 0 {
			util.S.Infof("push net %s peer %s: assigning IP", q.Cnn, q.PeerName)
			// assign an IP address chosen by me
			for _, ipNet := range cn.IPs {
				usedIPs := []net.IPNet{}
				for _, peer := range cn.Peers {
					usedIPs = append(usedIPs, central.ToIPNets(peer.AllowedIPs)...)
				}
				ip, err := util.AssignAddress((*net.IPNet)(&ipNet), usedIPs)
				if err != nil {
					return &api.PushS{
						S: &api.PushS_Overflow{
							Overflow: fmt.Sprint(err),
						},
					}, nil
				}
				peer.AllowedIPs = central.FromIPNets([]net.IPNet{{IP: ip, Mask: net.IPMask{0xff, 0xff, 0xff, 0xff}}})
			}
		}
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
	err = s.Tokens.AddToken(hash, TokenInfo{
		Name:     q.Name,
		Networks: q.Networks,
		CanPull:  q.CanPull,
		CanPush:  convCanPush(q.CanPush),
	}, q.Overwrite)
	if err != nil {
		return nil, err
	}
	return &api.AddTokenS{}, nil
}

func convCanPush(c *api.CanPush) *CanPush {
	if c == nil {
		return nil
	}
	return &CanPush{
		Networks: c.Networks,
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
