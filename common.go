package agent

import (
	"errors"
	"math/rand"
	"time"
)

var randSrc = rand.New(rand.NewSource(time.Now().UnixNano()))

// retryWithExponentialBackoff retries fn() up to n times at most, if fn() returns an error,
// then it returns nil right away.
// It applies exponential backoff in units of (1<<n) + jitter microsends.
func retryWithExponentialBackoff(nTries int64, timeBaseUnit time.Duration, fn func() error) (err error) {
	for i := int64(0); i < nTries; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		// Backoff for a time period with a pseudo-random jitter
		jitter := time.Duration(randSrc.Float64()*100) * time.Microsecond
		ts := jitter + ((1 << uint64(i)) * timeBaseUnit)
		<-time.After(ts)
	}
	return err
}

func preferedAddr(o *options) (string, error) {
	// NOTE: unix domain socket is preferred
	var preferred string
	if _, ok := o.addrs["tcp"]; ok {
		preferred = "tcp"
	}
	if _, ok := o.addrs["unix"]; ok {
		preferred = "unix"
	}

	if preferred == "" {
		return "", errors.New("find no addrs")
	}

	return preferred, nil
}
