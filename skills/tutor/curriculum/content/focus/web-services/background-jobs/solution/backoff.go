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
// retry boundary afterwards. This is "equal jitter" — half the delay fixed,
// half random, so the result is in [d/2, d] and never collapses to zero.
func (b Backoff) Delay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := b.Base
	for i := 1; i < attempt; i++ {
		if d >= b.Max {
			break
		}
		d *= 2
	}
	if b.Max > 0 && d > b.Max {
		d = b.Max
	}
	if d <= 0 {
		return 0
	}
	half := int64(d / 2)
	return time.Duration(half + b.random(half+1))
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
