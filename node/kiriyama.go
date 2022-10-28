package node

import (
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
	csLock sync.Mutex
	cs     map[int32]string
}

var _ api.KiriyamaServer = (*Kiriyama)(nil)

func newKiriyama(n *Node) *Kiriyama {
	return &Kiriyama{N: n, cs: map[int32]string{}}
}

func (k *Kiriyama) SetCS(i int, s string) {
	k.csLock.Lock()
	defer k.csLock.Unlock()
	k.cs[int32(i)] = s
}

func (k *Kiriyama) GetStatus(q *api.GetStatusQ, conn api.Kiriyama_GetStatusServer) error {
	k.csLock.Lock()
	defer k.csLock.Unlock()
	conn.Send(&api.GetStatusS{
		Cs: k.cs,
	})
	k.cs = map[int32]string{}
	return nil
}

func (k *Kiriyama) Loop() {
	server := grpc.NewServer()
	api.RegisterKiriyamaServer(server, k)
	lis, ok := kiriyamaSetup()
	if !ok {
		return
	}
	err := server.Serve(lis)
	if err != nil {
		util.S.Warnf("kiriyama serve: %s", err)
	}
}
