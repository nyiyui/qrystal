package mio

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/nyiyui/qrystal/runner"
)

func Listen() (lis net.Listener, addr string, err error) {
	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		err = fmt.Errorf("rand: %w", err)
		return
	}
	sockPath := filepath.Join(os.TempDir(), hex.EncodeToString(b)) + ".sock"
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
	u, err := user.Lookup(runner.NodeUser)
	if err != nil {
		return
	}
	gid, err = strconv.ParseInt(u.Gid, 10, 32)
	return
}

// Guard does nothing in linux as the connection is a socket, not via IP.
func Guard(h http.Handler) http.Handler { return h }
