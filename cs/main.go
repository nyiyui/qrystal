package cs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/nyiyui/qrystal/node"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
)

const commitDelay = 1 * time.Second

type CentralSource struct {
	api.UnimplementedCentralSourceServer
	notifyChs     map[string]chan change
	notifyChsLock sync.RWMutex
	cc            node.CentralConfig
	ccLock        sync.RWMutex
	tokens        tokenStore
	backportLock  sync.Mutex
	// keep it simple (RWMutex might be more appropriate but we're not writing
	// backports simultaneously (I hope))
	backportPath string
}

func New(cc node.CentralConfig, backportPath string) *CentralSource {
	return &CentralSource{
		notifyChs:    map[string]chan change{},
		cc:           cc,
		tokens:       newTokenStore(),
		backportPath: backportPath,
	}
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
	ti, ok := s.tokens.getToken(q.CentralToken)
	if !ok {
		return newTokenAuthError(q.CentralToken)
	}
	if !ti.CanPull {
		return errors.New("cannot pull")
	}
	log.Printf("%sから新たな認証済プル", ti.Name)
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
			ti, ok := s.tokens.getToken(q.CentralToken)
			if !ok {
				return newTokenAuthError(q.CentralToken)
			}
			if !ti.CanPull {
				return errors.New("cannot pull")
			}

			log.Printf("%sに送ります。", ti.Name)

			if ch.except == q.CentralToken {
				continue
			}

			newCC, err := s.convertCC(ti.Networks)
			if err != nil {
				log.Printf("convertCC: %s", err)
				return errors.New("conversion failed")
			}
			err = ss.Send(&api.PullS{Cc: newCC, ForwardingOnly: ch.forwardingOnly, Reason: ch.reason})
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
	var n2 *net.IPNet
	for i, n := range nets {
		_, n2, err = net.ParseCIDR(n.Cidr)
		if err != nil {
			return nil, err
		}
		dest[i] = *n2
	}
	return
}

func sliceToMap(ss []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}

func missingFromFirst(m1, m2 map[string]struct{}) []string {
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
		log.Printf("backport error: %s", err)
	} else {
		log.Printf("backport ok")
	}
	s.notifyChsLock.RLock()
	defer s.notifyChsLock.RUnlock()
	forwardsForPeers := make([]string, 0)
	for token, cnCh := range s.notifyChs {
		if token == ch.except {
			continue
		}
		ti, ok := s.tokens.getToken(token)
		if !ok {
			panic(fmt.Sprintf("getToken on token %s failed", token))
		}
		if ch.forwardingOnly {
			peersToForward := make([]string, 0, len(ch.forwardeePeers))
			for _, peerName := range ch.forwardeePeers {
				if peerName != ti.Name {
					// forwarding for itself is useless
					peersToForward = append(peersToForward, peerName)
				}
			}
			peer := s.cc.Networks[ch.net].Peers[ti.Name]
			util.S.Debugf("notifyChange net %s peer %s forwards for peer %s: peersToForward1: %s", ch.net, ch.peerName, ti.Name, peersToForward)
			peersToForward = missingFromFirst(sliceToMap(peer.ForwardingPeers), sliceToMap(peersToForward))
			util.S.Debugf("notifyChange net %s peer %s forwards for peer %s: peersToForward2: %s", ch.net, ch.peerName, ti.Name, peersToForward)
			if len(peersToForward) == 0 {
				continue
			} else {
				forwardsForPeers = append(forwardsForPeers, ti.Name)
			}
		}
		timer := time.NewTimer(1 * time.Second)
		select {
		case cnCh <- ch:
		case <-timer.C:
			util.S.Warnf("notifyChange net %s peer %s forwards for peer %s: chan send timeout", ch.net, ch.peerName, ti.Name)
		}
	}
	log.Printf("notifyChange net %s peer %s forwards for peers %s", ch.net, ch.peerName, forwardsForPeers)
}

func (s *CentralSource) Push(ctx context.Context, q *api.PushQ) (*api.PushS, error) {
	ti, ok := s.tokens.getToken(q.CentralToken)
	if !ok {
		return nil, newTokenAuthError(q.CentralToken)
	}
	if ti.CanPush == nil {
		return nil, errors.New("cannot push")
	}
	pattern, ok := ti.CanPush.Networks[q.Cnn]
	if !ok {
		return nil, fmt.Errorf("cannot push to net %s", q.Cnn)
	}
	if q.PeerName != pattern {
		return nil, fmt.Errorf("cannot push to net %s peer %s", q.Cnn, q.PeerName)
	}

	log.Printf("push %#v", q)

	peer, err := convertPeer(q.Peer)
	if err != nil {
		return &api.PushS{
			S: &api.PushS_InvalidData{
				InvalidData: fmt.Sprint(err),
			},
		}, nil
	}
	pushS, err := func() (*api.PushS, error) {
		s.ccLock.Lock()
		defer s.ccLock.Unlock()
		cn := s.cc.Networks[q.Cnn]
		if len(peer.AllowedIPs) == 0 {
			log.Printf("push net %s peer %s: assigning IP", q.Cnn, q.PeerName)
			// assign an IP address chosen by me
			for _, ipNet := range cn.IPs {
				usedIPs := []net.IPNet{}
				for _, peer := range cn.Peers {
					usedIPs = append(usedIPs, node.ToIPNets(peer.AllowedIPs)...)
				}
				ip, err := util.AssignAddress(&ipNet.IPNet, usedIPs)
				if err != nil {
					return &api.PushS{
						S: &api.PushS_Overflow{
							Overflow: fmt.Sprint(err),
						},
					}, nil
				}
				peer.AllowedIPs = node.FromIPNets([]net.IPNet{{IP: ip, Mask: net.IPMask{0xff, 0xff, 0xff, 0xff}}})
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
	log.Printf("push net %s peer %s: notify change", q.Cnn, q.PeerName)
	log.Printf("debug: %#v", s.cc.Networks[q.Cnn])
	log.Printf("debug: %#v", s.cc.Networks[q.Cnn].Peers[q.PeerName])
	s.notifyChange(change{reason: fmt.Sprintf("push net %s peer %s", q.Cnn, q.PeerName)})
	return &api.PushS{
		S: &api.PushS_Ok{},
	}, nil
}

func (s *CentralSource) AddToken(ctx context.Context, q *api.AddTokenQ) (*api.AddTokenS, error) {
	ti, ok := s.tokens.getToken(q.CentralToken)
	if !ok {
		return nil, newTokenAuthError(q.CentralToken)
	}
	if ti.CanAddTokens == nil {
		return nil, errors.New("cannot add tokens")
	}
	var hash sha256Sum
	n := copy(hash[:], q.Hash)
	if n != len(hash) {
		return nil, fmt.Errorf("hash %d length invalid (expected %d)", n, len(hash))
	}
	alreadyExists := s.tokens.AddToken(hash, TokenInfo{
		Name:    q.Name,
		CanPull: q.CanPull,
		CanPush: convCanPush(q.CanPush),
	}, q.Overwrite)
	if alreadyExists {
		return nil, errors.New("same hash already exists")
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

func (s *CentralSource) CanForward(ctx context.Context, q *api.CanForwardQ) (*api.CanForwardS, error) {
	ti, ok := s.tokens.getToken(q.CentralToken)
	if !ok {
		return nil, newTokenAuthError(q.CentralToken)
	}
	forwarderPeer, ok := ti.Networks[q.Network]
	if !ok {
		return nil, errors.New("bad cn")
	}
	log.Printf("%s can forward for %s", forwarderPeer, q.ForwardeePeers)
	cn := s.cc.Networks[q.Network]
	for _, forwardeePeer := range q.ForwardeePeers {
		peer := cn.Peers[forwardeePeer]
		peer.ForwardingPeers = append(peer.ForwardingPeers, forwarderPeer)
	}
	go s.notifyChange(change{reason: fmt.Sprintf("CanForward by %s", forwarderPeer), except: q.CentralToken, forwardingOnly: true, net: q.Network, forwardeePeers: q.ForwardeePeers, peerName: ti.Name})
	return &api.CanForwardS{}, nil
}
