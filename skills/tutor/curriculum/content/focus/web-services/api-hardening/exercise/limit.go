package harden

import (
	"net/http"
	"sync"
	"time"
)

// bucket is one client's token bucket. tokens is the balance as of updated;
// the tokens accrued since then are computed on read, so there is no
// background goroutine per client.
type bucket struct {
	tokens  float64
	updated time.Time
}

// Limiter is a per-key token bucket. Every key gets its own bucket that fills
// at rate tokens per second up to a ceiling of burst, and every allowed
// request spends one token.
//
// It is safe for concurrent use: the HTTP server hands each request its own
// goroutine, so every field here is guarded by mu.
type Limiter struct {
	mu      sync.Mutex
	clock   Clock
	rate    float64
	burst   float64
	buckets map[string]*bucket
}

// NewLimiter returns a Limiter that grants ratePerSecond tokens per second per
// key, with a ceiling of burst tokens. ratePerSecond must be greater than zero
// and burst at least one. A nil clock means RealClock.
func NewLimiter(ratePerSecond float64, burst int, clk Clock) *Limiter {
	if clk == nil {
		clk = RealClock{}
	}
	return &Limiter{
		clock:   clk,
		rate:    ratePerSecond,
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
	}
}

// Allow spends one token for key. It reports whether the request may proceed
// and, when it may not, how long the caller should wait before the next token
// exists.
func (l *Limiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	// TODO: read the time once from l.clock, then under l.mu:
	//   - find or create key's bucket (a new bucket starts full),
	//   - add rate * elapsed seconds of tokens, capped at burst,
	//   - if at least one token is available, spend it and allow,
	//   - otherwise report how long until the balance reaches 1.
	return true, 0
}

// Cleanup deletes every bucket untouched for at least idle and returns how
// many it removed. Without it the map grows once per distinct key seen, which
// is a memory exhaustion vector handed to anyone with a range of addresses.
//
// Choose idle no smaller than burst/rate, the time a drained bucket needs to
// refill completely: evicting a partly drained bucket hands its owner a full
// one, which is exactly the limit you were trying to enforce.
func (l *Limiter) Cleanup(idle time.Duration) int {
	// TODO: remove every bucket whose last update is at least idle in the
	// past, and return the count.
	return 0
}

// RateLimit rejects requests from a key that has run out of tokens with 429
// and a Retry-After header, and never calls next for them.
//
// key decides what "a client" means: the peer address, an API key, an account
// id. Pick the most specific thing you can trust.
func RateLimit(l *Limiter, key func(*http.Request) string) Middleware {
	// TODO: on refusal set Retry-After to whole seconds (RFC 9110 has no
	// finer unit), never below 1, and answer with WriteError and
	// http.StatusTooManyRequests. On success call next.
	return func(next http.Handler) http.Handler { return next }
}
