package pg

import (
	"time"
)

// ExpBackOff holds data used for exponential backoff.
type ExpBackOff struct {
	// Coefficient multiplied against attempt on each run to get exponential sleep duration.
	Coefficient int

	// MaxDelay maximum amount of time to sleep between retries.
	MaxDelay time.Duration

	// MaxRetries is the maximum number of times to trying the call before giving up.
	MaxRetries int

	// attempt is used internally to track how many times the function has been called recursively.
	attempt int
}

// Attempt increments the attempt counter and returns the value.
func (t *ExpBackOff) Attempt() int {
	a := t.attempt
	t.attempt++
	return a
}

// Delay is the next delay after the current attempt.
func (t *ExpBackOff) Delay() time.Duration {
	d := time.Duration(t.attempt*t.Coefficient) * time.Second
	if d > t.MaxDelay {
		return t.MaxDelay
	}

	return d
}
