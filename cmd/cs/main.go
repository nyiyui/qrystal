package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/nyiyui/qrystal/cs"
	"github.com/nyiyui/qrystal/profile"
	"github.com/nyiyui/qrystal/util"
	"github.com/tidwall/buntdb"
)

var configPath string

func main() {
	flag.StringVar(&configPath, "config", "", "config file path")
	flag.Parse()

	util.SetupLog()

	util.ShowCurrent()
	profile.Profile()

	config, err := cs.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("load config: %s", err)
	}
	log.Print("config file loaded")

	db, err := buntdb.Open(config.DBPath)
	if err != nil {
		log.Fatalf("open db: %s", err)
	}
	cs2, err := cs.New(*config.CC, config.BackportPath, db)
	if err != nil {
		log.Fatalf("new: %s", err)
	}
	err = warnNoPorts(config)
	if err != nil {
		log.Fatalf("warn no ports: %s", err)
	}
	err = warnDivergentTokens(config, cs2)
	if err != nil {
		log.Fatalf("warn divergent tokens: %s", err)
	}
	err = cs2.AddTokens(config.Tokens.Raw)
	if err != nil {
		log.Fatalf("add tokens: %s", err)
	}
	if config.BackportPath != "" && false {
		err = cs2.ReadBackport()
		if err != nil {
			log.Fatalf("read backport: %s", err)
		}
		log.Printf("read backport from %s", config.BackportPath)
	}
	err = cs2.Handle(config.Addr, config.TLS)
	if err != nil {
		log.Fatalf("Handle: %s", err)
	}
	err = cs2.HandleHaruka(config.HarukaAddr, config.TLS)
	if err != nil {
		log.Printf("HandleHaruka: %s", err)
	}
	err = util.Notify("READY=1\nSTATUS=serving…")
	if err != nil {
		log.Printf("Notify: %s", err)
	}
	log.Printf("notify ok")
	select {}
}

// warnDivergentTokens warns for any divergent tokens.
func warnDivergentTokens(config *cs.Config, server *cs.CentralSource) error {
	for _, tr := range config.Tokens.Raw {
		already, ok, err := server.Tokens.GetTokenByHash(hex.EncodeToString(tr.Hash[:]))
		if err != nil {
			return fmt.Errorf("get token %x: %s", tr.Hash[:], err)
		}
		if !ok {
			continue
		}
		info2, err := json.Marshal(tr.Info)
		if err != nil {
			return fmt.Errorf("marshal token2 %x: %s", tr.Hash[:], err)
		}
		already2, err := json.Marshal(already)
		if err != nil {
			return fmt.Errorf("marshal token2 %x: %s", tr.Hash[:], err)
		}
		if !bytes.Equal(info2, already2) {
			util.S.Warnf("token %x diverges from db", tr.Hash[:])
		}
	}
	return nil
}

func warnNoPorts(config *cs.Config) error {
	bad := false
	for cnn, cn := range config.CC.Networks {
		for pn, peer := range cn.Peers {
			if peer.Host != "" {
				_, _, err := net.SplitHostPort(peer.Host)
				if err != nil {
					util.S.Warnf("net %s peer %s has bad host: %s", cnn, pn, err)
					bad = true
				}
			}
		}
	}
	if bad {
		return errors.New("bad hosts")
	}
	return nil
}
