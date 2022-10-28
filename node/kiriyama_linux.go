package node

import (
	"net"
	"os"

	"github.com/nyiyui/qrystal/util"
)

func kiriyamaSetup() (lis net.Listener, ok bool) {
	err := os.Remove(kiriyamaAddr)
	if err != nil {
		util.S.Warnf("kiriyama cleanup: %s", err)
		return nil, false
	}
	util.S.Infof("kiriyama cleanup ok")
	lis, err = net.Listen("unix", kiriyamaAddr)
	if err != nil {
		util.S.Errorf("kiriyama listen: %s", err)
		return nil, false
	}
	os.Chmod(kiriyamaAddr, 0o770)
	// os.Chown(kiriyamaAddr)
	return nil, false
}

const kiriyamaAddr = "/run/qrystal-kiriyama/runner.sock"
