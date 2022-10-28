package tray

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/getlantern/systray"
	"github.com/nyiyui/qrystal/node/api"
	"github.com/nyiyui/qrystal/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var kc kiriyamaConn

func Main() {
	util.L, _ = zap.NewDevelopment()
	defer util.L.Sync()
	util.S = util.L.Sugar()

	util.ShowCurrent()

	var target string
	flag.StringVar(&target, "target", "passthrough:///unix:///run/qrystal-kiriyama/runner.sock", "kiriyama target")
	flag.Parse()

	if !strings.Contains(target, "unix:///") {
		util.S.Fatalf("only sockets supported")
	}

	clientConn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// NOTE: insecure conn is fine here because this should be protected by fs perms.
	if err != nil {
		util.S.Fatalf("dial target: %s", err)
	}
	kryCl := api.NewKiriyamaClient(clientConn)
	kc = kiriyamaConn{cl: kryCl}
	go kc.loop()
	systray.Run(onReady, onExit)
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
	tc := time.NewTicker(1 * time.Second)
	for range tc.C {
		s, err := conn.Recv()
		if err != nil {
			if err == io.EOF {
				return true, nil
			}
			util.S.Warnf("GetStatus Recv: %s", err)
			return false, fmt.Errorf("recv: %w", err)
		}
		util.S.Infof("GetStatus S: %s", s)
	}
	panic("unreacheable")
}

func onReady() {
	systray.SetTitle("Qrystal")
	systray.AddMenuItem("Status", "")
}

func onExit() {
}
