package util

import (
	"errors"
	"time"
)

// OnceTimeout is the general timeout used for network requests (the request inside a backoff once function).
const OnceTimeout = 10 * time.Second

const backoffMax = 5 * time.Minute

var ErrEndBackoff = errors.New("internal: end backoff")

func Backoff(once func() (resetBackoff bool, err error), singleError func(backoff time.Duration, err error) error) error {
	backoff := 1 * time.Second
	for {
		resetBackoff, err := once()
		if resetBackoff {
			backoff = 1 * time.Second
		}
		if err == nil {
			continue
		} else if err == ErrEndBackoff {
			return nil
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
