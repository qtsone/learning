package harden

import (
	"math"
	"net/http"
	"strconv"
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
	// Read the clock before taking the lock: Now may be a syscall, and the
	// lock is on the hot path of every request the process serves.
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, updated: now}
		l.buckets[key] = b
	} else if elapsed := now.Sub(b.updated); elapsed > 0 {
		b.tokens = min(l.burst, b.tokens+elapsed.Seconds()*l.rate)
		b.updated = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	if l.rate <= 0 {
		// A non-positive rate never refills; report a retry that will not
		// help rather than dividing by zero into an undefined Duration.
		return false, time.Duration(math.MaxInt64)
	}
	return false, time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
}

// Cleanup deletes every bucket untouched for at least idle and returns how
// many it removed. Without it the map grows once per distinct key seen, which
// is a memory exhaustion vector handed to anyone with a range of addresses.
//
// Choose idle no smaller than burst/rate, the time a drained bucket needs to
// refill completely: evicting a partly drained bucket hands its owner a full
// one, which is exactly the limit you were trying to enforce.
func (l *Limiter) Cleanup(idle time.Duration) int {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	removed := 0
	for key, b := range l.buckets {
		if now.Sub(b.updated) >= idle {
			delete(l.buckets, key)
			removed++
		}
	}
	return removed
}

// RateLimit rejects requests from a key that has run out of tokens with 429
// and a Retry-After header, and never calls next for them.
//
// key decides what "a client" means: the peer address, an API key, an account
// id. Pick the most specific thing you can trust.
func RateLimit(l *Limiter, key func(*http.Request) string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, retryAfter := l.Allow(key(r))
			if !ok {
				// Retry-After takes either whole seconds or an HTTP-date
				// (RFC 9110 §10.2.3); seconds is the one to use for a limiter.
				// Round up, never down: a rounded-down value invites the
				// client to retry into another refusal.
				seconds := int(math.Ceil(retryAfter.Seconds()))
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				WriteError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
