//go:build !sdnotify

package util

import "log"

func Notify(state string) (err error) {
	log.Printf("notify: %s", state)
	return nil
}
