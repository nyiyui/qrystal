package node

import (
	"net"
	"os"

	"github.com/nyiyui/qrystal/util"
)

func kiriyamaSetup() (lis net.Listener, ok bool) {
	err := os.Remove(kiriyamaAddr)
	if err != nil && !os.IsNotExist(err) {
		util.S.Errorf("kiriyama cleanup: %s", err)
		return nil, false
	}
	util.S.Infof("kiriyama cleanup ok")
	lis, err = net.Listen("unix", kiriyamaAddr)
	if err != nil {
		util.S.Errorf("kiriyama listen: %s", err)
		return nil, false
	}
	err = os.Chmod(kiriyamaAddr, 0o770)
	if err != nil {
		util.S.Errorf("kiriyama sock chmod: %s", err)
		return nil, false
	}
	util.S.Info("kiriyama setup ok")
	// os.Chown(kiriyamaAddr)
	return lis, true
}

const kiriyamaAddr = "/tmp/qrystal-kiriyama-runner.sock"
