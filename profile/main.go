package profile

import (
	"os"
	"runtime/pprof"
	"time"

	"github.com/nyiyui/qrystal/util"
)

func Profile() {
	profilePath := os.Getenv("QRYSTAL_PROFILE_PATH")
	if profilePath != "" {
		profile(profilePath)
	}
}

func profile(profilePath string) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			f, err := os.Open(profilePath)
			if err != nil {
				util.S.Warnf("profile open: %s", err)
				continue
			}
			defer func() {
				err := f.Close()
				if err != nil {
					util.S.Warnf("profile close: %s", err)
				}
			}()
			if err := pprof.WriteHeapProfile(f); err != nil {
				util.S.Warnf("profile write: %s", err)
			}
		}
	}()
}
