package jobs

import (
	"math/rand/v2"
	"time"
)

// Backoff turns an attempt number into how long to wait before the next one.
//
// Rand returns a value in [0,n) and exists so the jitter is injectable: a
// randomised test you cannot re-run is a coin flip, not a test. Leave it nil
// in production and the concurrency-safe global source is used.
type Backoff struct {
	Base time.Duration
	Max  time.Duration
	Rand func(n int64) int64
}

// Delay reports how long to wait after the given attempt number (1 is the
// first attempt).
//
// The exponential part spaces retries out so a struggling dependency is not
// hit at the same rate that just failed. The jitter part is what stops a
// thousand jobs that failed together from retrying together: identical
// backoff schedules turn one outage into a synchronised stampede at every
// retry boundary afterwards.
//
// The policy to implement is "equal jitter": take the nominal delay
// d = Base * 2^(attempt-1), capped at Max, and return a value in [d/2, d] —
// half fixed, half random, never collapsing to zero. Attempts below 1 count
// as the first attempt. Ask b.random for the random half.
func (b Backoff) Delay(attempt int) time.Duration {
	// TODO
	return 0
}

func (b Backoff) random(n int64) int64 {
	if n <= 0 {
		return 0
	}
	if b.Rand != nil {
		return b.Rand(n)
	}
	return rand.Int64N(n)
}
