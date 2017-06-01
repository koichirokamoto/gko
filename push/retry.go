package push

import (
	"time"

	"google.golang.org/api/gensupport"
)

// Retry function with backoff.
//
// It's reference google.golang.org/api/gensupport/retry.go
func retry(f func() error, backoff gensupport.BackoffStrategy) error {
	for {
		err := f()
		pause, retry := backoff.Pause()
		if !retry {
			return err
		}

		select {
		case <-time.After(pause):
		}
	}
}
