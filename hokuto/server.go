package hokuto

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/rpc"
	"os"
	"sync"

	"github.com/miekg/dns"
	"github.com/nyiyui/qrystal/central"
	"github.com/nyiyui/qrystal/hokuto/simple"
	"github.com/nyiyui/qrystal/mio"
	"github.com/nyiyui/qrystal/util"
)

var token []byte

var inited bool
var initedLock sync.Mutex

type Hokuto struct{}

type UpdateCCQ struct {
	Token []byte
	CC    *central.Config
}

func (_ Hokuto) UpdateCC(q *UpdateCCQ, _ *bool) error {
	if !bytes.Equal(token, q.Token) {
		return errors.New("token mismatch")
	}
	if q.CC == nil {
		return errors.New("cc is nil")
	}
	tmpSC := simple.ConvertFromCC(q.CC)
	util.S.Infof("UpdateCC: %s", sc)
	scLock.Lock()
	defer scLock.Lock()
	sc = &tmpSC
	return nil
}

type InitQ struct {
	Addr         string
	Parent       string
	ExtraParents []ExtraParent
}

// ExtraParent configuration.
type ExtraParent struct {
	Network string `yaml:"network"`
	Domain  string `yaml:"domain"`
}

func (_ Hokuto) Init(q *InitQ, _ *bool) (err error) {
	initedLock.Lock()
	defer initedLock.Unlock()
	if inited {
		return errors.New("already inited")
	}
	if q.Parent == "" {
		util.S.Warnf("parent is blank so using .qrystal.internal")
		q.Parent = ".qrystal.internal"
	}
	suffix = q.Parent + "."
	dns.HandleFunc(".", handle)
	server := &dns.Server{Addr: q.Addr, Net: "udp"}
	util.S.Infof("listening on %s", server.Addr)
	inited = true
	go func() {
		err = server.ListenAndServe()
		if err != nil {
			util.S.Fatalf("listen failed: %s\n ", err.Error())
		}
	}()
	return nil
}

type Mio struct{}

func (_ Mio) Ping(q string, r *string) error {
	*r = "pong"
	return nil
}

func Main() error {
	var tokenBase64 string
	var err error
	token, tokenBase64, err = mio.GenToken()
	if err != nil {
		util.S.Fatalf("GenToken: %s", err)
	}
	lis, addr, err := mio.Listen()
	if err != nil {
		util.S.Fatalf("Listen: %s", err)
	}
	fmt.Printf("addr:%s\n", addr)
	fmt.Printf("token:%s\n", tokenBase64)
	err = os.Stdout.Close()
	if err != nil {
		util.S.Fatalf("close stdout: %s", err)
	}
	util.S.Info("聞きます。")
	rs := rpc.NewServer()
	rs.Register(Hokuto{})
	rs.Register(Mio{})
	handler := mio.Guard(rs)
	util.S.Fatalf("serve: %s", http.Serve(lis, handler))
	return nil
}

func reverse(s []string) {
	switch len(s) {
	case 1:
	case 2:
		s[0], s[1] = s[1], s[0]
	default:
		for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
			s[i], s[j] = s[j], s[i]
		}
	}
}
