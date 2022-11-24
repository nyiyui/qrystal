package profile

import (
	"log"
	"os"
	"runtime/pprof"
	"time"
)

func Profile() {
	log.Print("profiling is enabled")
	profilePath := os.Getenv("QRYSTAL_PROFILE_PATH")
	if profilePath != "" {
		profile(profilePath)
	}
}

func profile(profilePath string) {
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
