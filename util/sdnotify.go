//go:build sdnotify

// The MIT License (MIT)
//
// Copyright (c) 2016 okzk
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Some code borrowed/stolen from <https://github.com/okzk/sdnotify/blob/d9becc38acbd785892af7637319e2c5e101057f7/notify_linux.go>.

package util

import (
	"errors"
	"fmt"
	"net"
	"os"
)

func Notify(state string) (err error) {
	name := os.Getenv("NOTIFY_SOCKET")
	if name == "" {
		return errors.New("sdnotify: NOTIFY_SOCKET not set")
	}
	var conn net.Conn
	conn, err = net.DialUnix("unixgram", nil, &net.UnixAddr{Name: name, Net: "unixgram"})
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
	S.Infof("notify: %s", state)
	return nil
}
