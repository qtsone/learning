package scalability

import (
	"math"
	"time"
)

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

// refill lazily credits tokens for the time elapsed since the last refill.
// Fractional tokens accumulate; the balance never exceeds capacity.
func (tb *TokenBucket) refill() {
	t := tb.now()
	if elapsed := t.Sub(tb.last).Seconds(); elapsed > 0 {
		tb.tokens = math.Min(tb.capacity, tb.tokens+elapsed*tb.rate)
		tb.last = t
	}
}

// Allow reports whether one request may proceed now, spending a token if so.
func (tb *TokenBucket) Allow() bool {
	tb.refill()
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// RetryAfter returns how long a denied caller must wait until a token is
// available, and 0 if one is available right now. Rounded up so that a
// caller who waits the returned duration is guaranteed a token.
func (tb *TokenBucket) RetryAfter() time.Duration {
	tb.refill()
	if tb.tokens >= 1 {
		return 0
	}
	deficit := 1 - tb.tokens
	return time.Duration(math.Ceil(deficit / tb.rate * float64(time.Second)))
}
