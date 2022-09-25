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
)

type CentralSource struct {
	api.UnimplementedCentralSourceServer
	notifyChs     map[string]chan change
	notifyChsLock sync.RWMutex
	cc            node.CentralConfig
	ccLock        sync.RWMutex
	tokens        tokenStore
}

func New(cc node.CentralConfig) *CentralSource {
	return &CentralSource{
		cc: cc,
	}
}

type change struct{}

var _ api.CentralSourceServer = new(CentralSource)

func (s *CentralSource) Ping(ctx context.Context, ss *api.PingQS) (*api.PingQS, error) {
	return &api.PingQS{}, nil
}

func (s *CentralSource) Pull(q *api.PullQ, ss api.CentralSource_PullServer) error {
	ti, ok := s.tokens.getToken(q.CentralToken)
	if !ok {
		return errors.New("token auth failed")
	}
	if !ti.CanPull {
		return errors.New("cannot pull")
	}
	log.Printf("%sから新たな認証済プル", ti.Name)
	// TODO: incremental changes (e.g. added peer) (instead of sending whole config every time)
	ctx := ss.Context()
	cnCh := s.addChangeNotify(q.CentralToken, 2)
	defer close(cnCh)
	cnCh <- change{}
	for {
		select {
		case <-ctx.Done():
			s.rmChangeNotify(q.CentralToken)
		case <-cnCh:
			// token status could change while this is called
			ti, ok := s.tokens.getToken(q.CentralToken)
			if !ok {
				return errors.New("token auth failed")
			}
			if !ti.CanPull {
				return errors.New("cannot pull")
			}

			log.Printf("%sに送ります。", ti.Name)

			s.ccLock.RLock()
			defer s.ccLock.RUnlock()
			newCC, err := s.convertCC(ti.Networks)
			if err != nil {
				log.Printf("convertCC: %s", err)
				return errors.New("conversion failed")
			}
			err = ss.Send(&api.PullS{Cc: newCC})
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

func (s *CentralSource) notifyChange() {
	s.notifyChsLock.RLock()
	defer s.notifyChsLock.RUnlock()
	for _, cnCh := range s.notifyChs {
		go func(cnCh chan change) {
			timer := time.NewTimer(1 * time.Second)
			select {
			case cnCh <- change{}:
			case <-timer.C:
			}
		}(cnCh)
	}
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
	s.notifyChange()
	return &api.PushS{
		S: &api.PushS_Ok{},
	}, nil
}

func (s *CentralSource) AddToken(ctx context.Context, q *api.AddTokenQ) (*api.AddTokenS, error) {
	ti, ok := s.tokens.getToken(q.CentralToken)
	if !ok {
		return nil, errors.New("token auth failed")
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
		Name:     q.Name,
		Networks: q.Networks,
		CanPull:  q.CanPull,
		CanPush:  q.CanPush,
	}, q.Overwrite)
	if alreadyExists {
		return nil, errors.New("same hash already exists")
	}
	return &api.AddTokenS{}, nil
}
