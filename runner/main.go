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

	mh, hh, nh, err := main2()
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
	if hh != nil {
		log.Print("stopping hokuto")
		err = hh.Cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Printf("hokuto: %s", err)
		}
		log.Print("stopped hokuto")
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

func main2() (mh, hh *mioHandle, nh *nodeHandle, err error) {
	cfg := newConfig()
	log.Printf("%#v", cfg)
	mh, err = newMio(&cfg.Mio)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("mio: %s", err)
	}
	log.Print("mio started")
	hh, err = newMio(&cfg.Hokuto)
	if err != nil {
		return mh, nil, nil, fmt.Errorf("hokuto: %s", err)
	}
	log.Print("hokuto started")
	nh, err = newNode(&cfg.Node, mh, hh)
	if err != nil {
		return mh, hh, nil, fmt.Errorf("node: %s", err)
	}
	_ = nh
	log.Print("node started")
	return mh, hh, nh, nil
}
