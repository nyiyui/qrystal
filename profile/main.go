package profile

import (
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/nyiyui/qrystal/util"
)

func Profile() {
	log.Print("profiling is enabled")
	enable := os.Getenv("QRYSTAL_PROFILE")
	if enable == "on" {
		profile()
	}
}

func profile() {
	f, err := os.CreateTemp("", "qrystal-profile-*")
	if err != nil {
		util.S.Errorf("create temp: %s", err)
		return
	}
	profilePath := f.Name()
	err = f.Close()
	if err != nil {
		util.S.Errorf("close temp: %s", err)
		return
	}
	writeProfile(profilePath)
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			writeProfile(profilePath)
		}
	}()
}

func writeProfile(profilePath string) {
	f, err := os.Create(profilePath)
	if err != nil {
		log.Printf("profile open: %s", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Printf("profile close: %s", err)
		}
	}()
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Printf("profile write: %s", err)
	}
}
