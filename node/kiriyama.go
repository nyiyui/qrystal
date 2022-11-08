package node

import (
	"net"
	"sync"

	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"google.golang.org/grpc"
)

type Kiriyama struct {
	// static
	N *Node
	api.UnimplementedKiriyamaServer

	// config
	Addr string

	// state
	csLock   sync.RWMutex
	cs       map[int32]*api.CSStatus
	peerLock sync.RWMutex
	peer     map[string]string
}

var _ api.KiriyamaServer = (*Kiriyama)(nil)

func newKiriyama(n *Node) *Kiriyama {
	return &Kiriyama{N: n, cs: map[int32]*api.CSStatus{}, peer: map[string]string{}}
}

func (k *Kiriyama) SetCS(i int, s string) {
	k.csLock.Lock()
	defer k.csLock.Unlock()
	util.S.Infof("kiriyama SetCS: %d %s", i, s)
	csc := k.N.cs[i]
	css := &api.CSStatus{
		Status: s,
	}
	if csc.Comment != "" {
		css.Name = csc.Comment
	} else {
		css.Name, _, _ = net.SplitHostPort(csc.Host)
	}
	k.cs[int32(i)] = css
}

func (k *Kiriyama) SetPeer(cnn string, pn string, s string) {
	k.peerLock.Lock()
	defer k.peerLock.Unlock()
	util.S.Infof("kiriyama SetPeer: %s %s %s", cnn, pn, s)
	k.peer[cnn+" "+pn] = s
}

func (k *Kiriyama) GetStatus(q *api.GetStatusQ, conn api.Kiriyama_GetStatusServer) error {
	k.csLock.RLock()
	defer k.csLock.RUnlock()
	conn.Send(&api.GetStatusS{
		Cs:   k.cs,
		Peer: k.peer,
	})
	return nil
}

func (k *Kiriyama) Loop() {
	server := grpc.NewServer()
	api.RegisterKiriyamaServer(server, k)
	lis, ok := kiriyamaSetup()
	if !ok {
		util.S.Errorf("kiriyama setup failed")
		return
	}
	util.S.Info("kiriyama serving")
	err := server.Serve(lis)
	if err != nil {
		util.S.Errorf("kiriyama serve: %s", err)
		return
	}
}
