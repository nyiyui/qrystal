//go:build sdnotify

package util

import (
	"errors"
	"fmt"
	"net"
	"os"
)

func Notify(state string) (err error) {
	sockAddr := os.Getenv("NOTIFY_SOCKET")
	if sockAddr == "" {
		return errors.New("sdnotify: NOTIFY_SOCKET not set")
	}
	var conn net.Conn
	conn, err = net.Dial("unixgram", sockAddr)
	if err != nil {
		return fmt.Errorf("sdnotify: Dial: %w", err)
	}
	defer func() {
		err2 := conn.Close()
		if err2 != nil {
			err = fmt.Errorf("sdnotify: Close: %w", err2)
		}
	}()
	_, err = conn.Write([]byte(state))
	if err != nil {
		return fmt.Errorf("sdnotify: Write: %w", err)
	}
	return nil
}
