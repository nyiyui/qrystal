package util

import (
	"log"
	"os/user"
)

func ShowCurrent() {
	u, err := user.Current()
	if err != nil {
		log.Printf("current: error: %s", err)
		return
	}
	log.Printf("current: uid %s gid %s", u.Uid, u.Gid)
}
