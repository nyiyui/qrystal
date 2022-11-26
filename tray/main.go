package tray

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var kc kiriyamaConn
var target string

func Main() {
	util.L, _ = zap.NewDevelopment()
	defer util.L.Sync()
	util.S = util.L.Sugar()

	util.ShowCurrent()

	flag.StringVar(&target, "target", "passthrough:///unix:///tmp/qrystal-kiriyama-runner.sock", "kiriyama target")
	flag.Parse()

	if !strings.Contains(target, "unix:///") {
		util.S.Fatalf("only sockets supported")
	}
}

type kiriyamaConn struct {
	cl api.KiriyamaClient
}

func (k *kiriyamaConn) loop() {
	util.S.Fatalf("loop2: %s", util.Backoff(k.loopOnce, func(backoff time.Duration, err error) error {
		util.S.Warnf("(backoff %s) loop: %s", backoff, err)
		return nil
	}))
}

func (k *kiriyamaConn) loopOnce() (resetBackoff bool, err error) {
	conn, err := kc.cl.GetStatus(context.Background(), &api.GetStatusQ{})
	if err != nil {
		return false, fmt.Errorf("recv: %w", err)
	}
	res := new(strings.Builder)
	s, err := conn.Recv()
	if err != nil {
		if err == io.EOF {
			return true, nil
		}
		util.S.Warnf("GetStatus Recv: %s", err)
		return false, fmt.Errorf("recv: %w", err)
	}
	util.S.Infof("GetStatus S: %s", s)
	for _, status := range s.Cs {
		fmt.Fprintf(res, "%s - %s\n", status.Name, status.Status)
	}
	for cnpn, status := range s.Peer {
		fmt.Fprintf(res, "%s - %s\n", cnpn, status)
	}
	fmt.Println(res)
	return false, util.ErrEndBackoff
}

func onReady() {
	clientConn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// NOTE: insecure conn is fine here because this should be protected by fs perms.
	if err != nil {
		util.S.Fatalf("dial target: %s", err)
	}
	kryCl := api.NewKiriyamaClient(clientConn)
	kc = kiriyamaConn{cl: kryCl}
	kc.loop()
}

func onExit() {
}
