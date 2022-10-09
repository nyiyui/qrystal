package node

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"

	"github.com/nyiyui/qrystal/node/api"
)

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

func (a *azusa) setup(n *Node, cl api.CentralSourceClient) error {
	log.Print("azusa: locking ccLock")
	for cnn, peerName := range a.networks {
		log.Printf("azusa: net %s peer %s: pushing", cnn, peerName)
		pubKey := n.coordPrivKey.Public().(ed25519.PublicKey)
		q := api.PushQ{
			CentralToken: n.csToken,
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
		log.Printf("azusa: net %s peer %s: pushed", cnn, peerName)
	}
	return nil
}
