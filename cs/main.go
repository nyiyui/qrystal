package cs

import (
	"context"
	"errors"
	"fmt"
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
	notifyChs     map[string]chan change
	notifyChsLock sync.RWMutex
	cc            central.Config
	ccLock        sync.RWMutex
	Tokens        TokenStore
	backportLock  sync.Mutex
	// keep it simple (RWMutex might be more appropriate but we're not writing
	// backports simultaneously (I hope))
	backportPath string
}

func New(cc central.Config, backportPath string, db *buntdb.DB) (*CentralSource, error) {
	ts, err := newTokenStore(db)
	return &CentralSource{
		notifyChs:    map[string]chan change{},
		cc:           cc,
		Tokens:       ts,
		backportPath: backportPath,
	}, err
}

type change struct {
	reason         string
	except         string
	forwardingOnly bool
	net            string
	forwardeePeers []string
	peerName       string
}

var _ api.CentralSourceServer = new(CentralSource)

func (s *CentralSource) Ping(ctx context.Context, ss *api.PingQS) (*api.PingQS, error) {
	return &api.PingQS{}, nil
}

func (s *CentralSource) Pull(q *api.PullQ, ss api.CentralSource_PullServer) error {
	ti, ok, err := s.Tokens.getToken(q.CentralToken)
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
	err = s.Tokens.UpdateToken(ti)
	if err != nil {
		return err
	}
	defer func() {
		ti.StopUse()
		err = s.Tokens.UpdateToken(ti)
		if err != nil {
			util.S.Errorf("UpdateToken %s: %s", ti.sum, err)
		}
	}()
	util.S.Infof("%sから新たな認証済プル", ti.Name)
	// TODO: incremental changes (e.g. added peer) (instead of sending whole config every time)
	ctx := ss.Context()
	cnCh := s.addChangeNotify(q.CentralToken, 2)
	defer close(cnCh)
	cnCh <- change{reason: "local"}
	for {
		select {
		case <-ctx.Done():
			s.rmChangeNotify(q.CentralToken)
		case ch := <-cnCh:
			// token status could change while this is called
			ti, ok, err := s.Tokens.getToken(q.CentralToken)
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

			if ch.except == q.CentralToken {
				continue
			}

			newCC, err := s.convertCC(ti.Networks)
			if err != nil {
				util.S.Infof("convertCC: %s", err)
				return errors.New("conversion failed")
			}
			var changedCNs []string
			if ch.net != "" {
				changedCNs = []string{ch.net}
			}
			err = ss.Send(&api.PullS{Cc: newCC, ForwardingOnly: ch.forwardingOnly, ChangedCNs: changedCNs, Reason: ch.reason})
			if err != nil {
				return err
			}
		}
	}
}

func (s *CentralSource) addChangeNotify(name string, chLen int) chan change {
	ch := make(chan change, chLen)
	s.notifyChsLock.Lock()
	defer s.notifyChsLock.Unlock()
	s.notifyChs[name] = ch
	return ch
}

func (s *CentralSource) rmChangeNotify(name string) {
	s.notifyChsLock.Lock()
	defer s.notifyChsLock.Unlock()
	delete(s.notifyChs, name)
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

func (s *CentralSource) notifyChange(ch change) {
	time.Sleep(commitDelay)
	err := s.backport()
	if err != nil {
		util.S.Warnf("backport error: %s", err)
	} else {
		util.S.Infof("backport ok")
	}
	s.notifyChsLock.RLock()
	defer s.notifyChsLock.RUnlock()
	forwardsForPeers := make([]string, 0)
	for token, cnCh := range s.notifyChs {
		if token == ch.except {
			continue
		}
		ti, ok, err := s.Tokens.getToken(token)
		if err != nil {
			util.S.Warnf("notifyChange net %s peer %s token: %s", ch.net, ch.peerName, err)
			continue
		}
		if !ok {
			panic(fmt.Sprintf("getToken on token %s failed", token))
		}
		if ch.forwardingOnly {
			peerName, ok := ti.Networks[ch.net]
			if !ok {
				util.S.Errorf("notifyChange net %s peer %s token %s hash %s not found", ch.net, peerName, ti.Name, token)
			}
			peersToForward := make([]string, 0, len(ch.forwardeePeers))
			for _, forwardee := range ch.forwardeePeers {
				if forwardee != peerName {
					// forwarding for itself is useless
					peersToForward = append(peersToForward, forwardee)
				}
			}
			peer, ok := s.cc.Networks[ch.net].Peers[peerName]
			if !ok {
				util.S.Warnf("notifyChange net %s peer %s not found in cc", ch.net, peerName)
				continue
			}
			util.S.Debugf("notifyChange net %s peer %s forwards for peer %s: peersToForward1: %s", ch.net, ch.peerName, peerName, peersToForward)
			peersToForward = MissingFromFirst(SliceToMap(peer.ForwardingPeers), SliceToMap(peersToForward))
			util.S.Debugf("notifyChange net %s peer %s forwards for peer %s: peersToForward2: %s", ch.net, ch.peerName, peerName, peersToForward)
			if len(peersToForward) == 0 {
				continue
			} else {
				forwardsForPeers = append(forwardsForPeers, peerName)
			}
		}
		timer := time.NewTimer(1 * time.Second)
		select {
		case cnCh <- ch:
		case <-timer.C:
			util.S.Warnf("notifyChange net %s peer %s forwards for peer %s: chan send timeout", ch.net, ch.peerName, ti.Name)
		}
	}
	util.S.Infof("notifyChange net %s peer %s forwards for peers %s %t", ch.net, ch.peerName, forwardsForPeers, ch.forwardingOnly)
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
	s.notifyChange(change{reason: fmt.Sprintf("push net %s peer %s", q.Cnn, q.PeerName)})
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

func (s *CentralSource) CanForward(ctx context.Context, q *api.CanForwardQ) (*api.CanForwardS, error) {
	if len(q.ForwardeePeers) == 0 {
		return nil, errors.New("no ForwardeePeers")
	}
	ti, ok, err := s.Tokens.getToken(q.CentralToken)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, newTokenAuthError(q.CentralToken)
	}
	forwarderPeer, ok := ti.Networks[q.Network]
	if !ok {
		return nil, errors.New("bad cn")
	}
	ti.Use()
	err = s.Tokens.UpdateToken(ti)
	if err != nil {
		return nil, err
	}
	util.S.Infof("%s can forward for %s", forwarderPeer, q.ForwardeePeers)
	cn := s.cc.Networks[q.Network]
	for _, forwardeePeer := range q.ForwardeePeers {
		peer := cn.Peers[forwardeePeer]
		if !contains(peer.ForwardingPeers, forwarderPeer) {
			peer.ForwardingPeers = append(peer.ForwardingPeers, forwarderPeer)
		}
	}
	go s.notifyChange(change{reason: fmt.Sprintf("CanForward by %s", forwarderPeer), except: q.CentralToken, forwardingOnly: true, net: q.Network, forwardeePeers: q.ForwardeePeers, peerName: ti.Name})
	return &api.CanForwardS{}, nil
}
