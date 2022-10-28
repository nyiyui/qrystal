package util

import (
	"time"
)

const backoffMax = 5 * time.Minute

func Backoff(once func() (resetBackoff bool, err error), singleError func(backoff time.Duration, err error) error) error {
	backoff := 1 * time.Second
	for {
		resetBackoff, err := once()
		if resetBackoff {
			backoff = 1 * time.Second
		}
		if err == nil {
			continue
		}
		err = singleError(backoff, err)
		if err != nil {
			return err
		}
		time.Sleep(backoff)
		backoff *= 2
		if resetBackoff {
			backoff = 1 * time.Second
		}
		if backoff > backoffMax {
			backoff = backoffMax
		}
	}
}
