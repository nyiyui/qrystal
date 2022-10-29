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

	systray.Run(onReady, onExit)
}

type kiriyamaConn struct {
	cl        api.KiriyamaClient
	csMenus   map[int32]*systray.MenuItem
	peerMenus map[string]*systray.MenuItem
}

func (k *kiriyamaConn) loop() {
	util.S.Fatalf("loop2: %s", util.Backoff(k.loopOnce, func(backoff time.Duration, err error) error {
		for _, menu := range k.csMenus {
			menu.SetTitle(fmt.Sprintf("backoff %s", backoff))
		}
		util.S.Warnf("(backoff %s) loop: %s", backoff, err)
		return nil
	}))
}

func (k *kiriyamaConn) loopOnce() (resetBackoff bool, err error) {
	conn, err := kc.cl.GetStatus(context.Background(), &api.GetStatusQ{})
	if err != nil {
		return false, fmt.Errorf("recv: %w", err)
	}
	tc := time.NewTicker(500 * time.Millisecond)
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
		for key, status := range s.Cs {
			title := fmt.Sprintf("%s - %s", status.Name, status.Status)
			if _, ok := k.csMenus[key]; ok {
				k.csMenus[key].SetTitle(title)
			} else {
				k.csMenus[key] = systray.AddMenuItem(title, "")
			}
		}
		for cnpn, status := range s.Peer {
			title := fmt.Sprintf("%s - %s", cnpn, status)
			if _, ok := k.peerMenus[cnpn]; ok {
				k.peerMenus[cnpn].SetTitle(title)
			} else {
				k.peerMenus[cnpn] = systray.AddMenuItem(title, "")
			}
		}
	}
	panic("unreacheable")
}

func onReady() {
	systray.SetTitle("Qrystal")
	clientConn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// NOTE: insecure conn is fine here because this should be protected by fs perms.
	if err != nil {
		util.S.Fatalf("dial target: %s", err)
	}
	kryCl := api.NewKiriyamaClient(clientConn)
	kc = kiriyamaConn{cl: kryCl, csMenus: map[int32]*systray.MenuItem{}, peerMenus: map[string]*systray.MenuItem{}}
	go kc.loop()
}

func onExit() {
}
