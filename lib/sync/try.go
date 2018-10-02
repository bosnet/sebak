package sync

import "errors"

type TryFunc func(attempt int) (retry bool, err error)

const MaxRetries = 10

var errMaxRetriesReached = errors.New("exceeded retry limit")

func Try(maxRetries int, fn TryFunc) error {
	var err error
	var cont bool
	attempt := 1
	for {
		cont, err = fn(attempt)
		if !cont || err == nil {
			break
		}
		attempt++
		// if maxRetries < 0 , infinite retries.
		if maxRetries > 0 && attempt > maxRetries {
			return errMaxRetriesReached
		}
	}
	return err
}

func TryForever(fn TryFunc) error {
	return Try(-1, fn)
}
