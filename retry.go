package gko

import (
	"errors"
	"time"

	"google.golang.org/api/gensupport"
)

// ErrNoRetry is error not needed to retry.
var ErrNoRetry = errors.New("no retry error")

// Retry retry function with backoff.
//
// It's reference google.golang.org/api/gensupport/retry.go
func Retry(f func() error, backoff gensupport.BackoffStrategy) error {
	for {
		err := f()
		pause, retry := backoff.Pause()
		if !shouldRetry(err) || !retry {
			return err
		}

		select {
		case <-time.After(pause):
		}
	}
}

// shouldRetry returns true if the HTTP response / error indicates that the
// request should be attempted again.
func shouldRetry(err error) bool {
	if err == ErrNoRetry {
		return false
	}
	return true
}
