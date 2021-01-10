// Package resilience provides helpers for dealing with resilience.
package resilience

import (
	"errors"
	"testing"
	"time"
)

// Retry executes a f until no error is returned or failAfter is reached.
func Retry(t *testing.T, maxWait, failAfter time.Duration, f func() error) (err error) {
	var lastStart time.Time
	err = errors.New("did not connect")
	loopWait := time.Millisecond * 100
	retryStart := time.Now().UTC()
	for retryStart.Add(failAfter).After(time.Now().UTC()) {
		lastStart = time.Now().UTC()
		if err = f(); err == nil {
			return nil
		}

		if lastStart.Add(maxWait * 2).Before(time.Now().UTC()) {
			retryStart = time.Now().UTC()
		}

		t.Logf("Error: %s. Retrying in %f seconds...", err, loopWait.Seconds())
		time.Sleep(loopWait)
		loopWait *= time.Duration(int64(2))
		if loopWait > maxWait {
			loopWait = maxWait
		}
	}
	return err
}
