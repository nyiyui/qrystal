package node

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"sync"

	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
)

type AzusaConfig struct {
	Host     string
	Networks map[string]string
}

func newAzusa(c AzusaConfig) *azusa {
	return &azusa{
		enabled:  true,
		host:     c.Host,
		networks: c.Networks,
	}
}

type azusa struct {
	enabled  bool
	networks map[string]string
	host     string
}

func (n *Node) AzusaConfigure(networks map[string]string, host string) {
	n.azusa.enabled = true
	n.azusa.networks = networks
	n.azusa.host = host
}

func (a *azusa) setup(n *Node, csc CSConfig, cl api.CentralSourceClient) error {
	errs := make([]error, len(a.networks))
	cnns := make([]string, len(a.networks))
	i := 0
	var wg sync.WaitGroup
	for cnn, peerName := range a.networks {
		cnns[i] = cnn
		wg.Add(1)
		go func(i int, cnn, peerName string) {
			defer wg.Done()
			errs[i] = a.setupOne(n, csc, cl, cnn, peerName)
			i++
		}(i, cnn, peerName)
	}
	wg.Wait()
	fail := false
	for i, err := range errs {
		cnn := cnns[i]
		if err != nil {
			util.S.Errorf("azusa: net %s peer %s error: %s", cnn, a.networks[cnn], err)
			fail = true
		}
	}
	if fail {
		return errors.New("failed; see logs")
	}
	return nil
}
func (a *azusa) setupOne(n *Node, csc CSConfig, cl api.CentralSourceClient, cnn, peerName string) error {
	util.S.Debugf("azusa: net %s peer %s: pushing", cnn, peerName)
	pubKey := n.coordPrivKey.Public().(ed25519.PublicKey)
	q := api.PushQ{
		CentralToken: csc.Token,
		Cnn:          cnn,
		PeerName:     peerName,
		Peer: &api.CentralPeer{
			Host:      a.host,
			PublicKey: &api.PublicKey{Raw: []byte(pubKey)},
		},
	}
	s, err := cl.Push(context.Background(), &q)
	if err != nil {
		return err
	}
	switch s := s.S.(type) {
	case *api.PushS_InvalidData:
		return fmt.Errorf("invalid data: %s", s.InvalidData)
	case *api.PushS_Overflow:
		return fmt.Errorf("overflow: %s", s.Overflow)
	case *api.PushS_Ok:
	default:
		panic(fmt.Sprintf("%#v", s))
	}
	util.S.Infof("azusa: net %s peer %s: pushed", cnn, peerName)
	return nil
}
