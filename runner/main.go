//go:build linux

package runner

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Main runs mio and node.
// NOTE: this program runs as a priviledged user.
func Main() {
	termCh := make(chan os.Signal, 2)
	signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM)

	mh, nh, err := main2()
	if err != nil {
		log.Printf("error: %s", err)
		termCh <- syscall.SIGTERM
	}

	<-termCh
	if mh != nil {
		log.Print("stopping mio")
		err = mh.Cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Printf("mio: %s", err)
		}
		log.Print("stopped mio")
	}
	if nh != nil {
		log.Print("stopping node")
		err = nh.Cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Printf("node: %s", err)
		}
		log.Print("stopped node")
	}
}

func main2() (mh *mioHandle, nh *nodeHandle, err error) {
	cfg := newConfig()
	log.Printf("%#v", cfg)
	mh, err = newMio(&cfg.Mio)
	if err != nil {
		return nil, nil, fmt.Errorf("mio: %s", err)
	}
	log.Print("mio started")
	nh, err = newNode(&cfg.Node, mh)
	if err != nil {
		return mh, nil, fmt.Errorf("mio: %s", err)
	}
	_ = nh
	log.Print("node started")
	return mh, nh, nil
}
