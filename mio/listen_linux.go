package mio

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/user"
	"strconv"

	"github.com/nyiyui/qrystal/runner"
)

const sockPath = "/tmp/qrystal-mio.sock"

func listen() (lis net.Listener, addr string, err error) {
	err = os.Remove(sockPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		err = fmt.Errorf("削除: %w", err)
		return
	}
	lis, err = net.Listen("unix", sockPath)
	if err != nil {
		err = fmt.Errorf("バインド: %w", err)
		return
	}
	err = os.Chmod(sockPath, 0o660)
	if err != nil {
		err = fmt.Errorf("chmod: %w", err)
		return
	}
	uid, err := getUid()
	if err != nil {
		err = fmt.Errorf("getUid: %w", err)
		return
	}
	gid, err := getGid()
	if err != nil {
		err = fmt.Errorf("getGid: %w", err)
		return
	}
	err = os.Chown(sockPath, int(uid), int(gid))
	if err != nil {
		err = fmt.Errorf("chmod: %w", err)
		return
	}
	addr = "unix:" + sockPath
	return
}

func getUid() (uid int64, err error) {
	u, err := user.Current()
	if err != nil {
		return
	}
	uid, err = strconv.ParseInt(u.Uid, 10, 32)
	return
}

func getGid() (gid int64, err error) {
	u, err := user.Lookup(runner.QrystalNodeUsername)
	if err != nil {
		return
	}
	gid, err = strconv.ParseInt(u.Gid, 10, 32)
	return
}

func guard(h http.Handler) http.Handler { return h }
