package scalability

import "time"

// TokenBucket is a rate limiter. The bucket holds at most capacity tokens,
// refills continuously at refillPerSec tokens per second, and each allowed
// request spends one token. It starts full, so a fresh client may burst up
// to capacity requests before the sustained rate applies.
//
// The current time is injected via now so tests control the clock — the
// same trick the caching and message-queues exercises used.
type TokenBucket struct {
	capacity float64
	rate     float64
	now      func() time.Time

	tokens float64
	last   time.Time
}

func NewTokenBucket(capacity int, refillPerSec float64, now func() time.Time) *TokenBucket {
	return &TokenBucket{
		capacity: float64(capacity),
		rate:     refillPerSec,
		now:      now,
		tokens:   float64(capacity),
		last:     now(),
	}
}

// Allow reports whether one request may proceed now, spending a token if so.
func (tb *TokenBucket) Allow() bool {
	// TODO: refill lazily — add elapsed-since-last-refill × rate tokens,
	// capped at capacity — then spend one token if at least one is available.
	return false
}

// RetryAfter returns how long a denied caller must wait until a token is
// available, and 0 if one is available right now. Round up: after waiting
// the returned duration, a request must succeed.
func (tb *TokenBucket) RetryAfter() time.Duration {
	// TODO: refill, then convert the token deficit into a wait at the
	// refill rate.
	return 0
}
