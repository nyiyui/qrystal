package cs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/nyiyui/qanms/node"
	"github.com/nyiyui/qanms/node/api"
)

type CentralSource struct {
	api.UnimplementedCentralSourceServer
	changeNotify     []chan change
	changeNotifyLock sync.Mutex
	cc               node.CentralConfig
	ccLock           sync.RWMutex
	tokens           tokenStore
}
type change struct{}

var _ api.CentralSourceServer = new(CentralSource)

func (s *CentralSource) Pull(q *api.PullQ, ss api.CentralSource_PullServer) error {
	ti, ok := s.tokens.getToken(q.CentralToken)
	if !ok {
		return errors.New("token auth failed")
	}
	if !ti.CanPull {
		return errors.New("cannot pull")
	}
	s.ccLock.RLock()
	defer s.ccLock.RUnlock()
	// TODO: incremental changes (e.g. added peer) (instead of sending whole config every time)
	cnCh := s.addChangeNotify(1)
	cnCh <- change{}
	select {
	case <-cnCh:
		s, err := s.convertCC(ti.Name)
		if err != nil {
			log.Printf("convertCC: %s", err)
			return errors.New("conversion failed")
		}
		err = ss.Send(&api.PullS{Cc: s})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *CentralSource) addChangeNotify(chLen int) chan change {
	ch := make(chan change, chLen)
	s.changeNotifyLock.Lock()
	defer s.changeNotifyLock.Unlock()
	s.changeNotify = append(s.changeNotify, ch)
	return ch
}

func fromIPNets(nets []net.IPNet) (dest []*api.IPNet) {
	dest = make([]*api.IPNet, len(nets))
	for i, n := range nets {
		dest[i] = &api.IPNet{Cidr: n.String()}
	}
	return
}

func toIPNets(nets []*api.IPNet) (dest []net.IPNet, err error) {
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

func (s *CentralSource) Push(ctx context.Context, q *api.PushQ) (*api.PushS, error) {
	ti, ok := s.tokens.getToken(q.CentralToken)
	if !ok {
		return nil, errors.New("token auth failed")
	}
	if !ti.CanPush {
		return nil, errors.New("cannot push")
	}
	peer, err := convertPeer(q.Peer)
	if err != nil {
		return &api.PushS{
			S: &api.PushS_InvalidData{
				InvalidData: fmt.Sprint(err),
			},
		}, nil
	}
	s.ccLock.Lock()
	defer s.ccLock.Unlock()
	cn := s.cc.Networks[q.Cnn]
	// TODO: impl checks for PublicKey, host, net overlap
	cn.Peers[q.PeerName] = peer
	return &api.PushS{
		S: &api.PushS_Ok{},
	}, nil
}
