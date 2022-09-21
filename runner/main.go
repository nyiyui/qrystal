//go:build linux

package runner

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nyiyui/qanms/runner/config"
	"gopkg.in/yaml.v3"
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
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()

	var cfg config.Root
	raw, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read config: %s", err)
	}
	err = yaml.Unmarshal(raw, &cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("decode config: %s", err)
	}
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
