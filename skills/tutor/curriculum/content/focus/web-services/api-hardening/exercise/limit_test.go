package harden

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestLimiterSpendsTheBurstThenRefuses(t *testing.T) {
	clk := newFakeClock()
	l := NewLimiter(1, 3, clk)

	for i := 1; i <= 3; i++ {
		if ok, _ := l.Allow("client-a"); !ok {
			t.Fatalf("request %d denied: a fresh bucket starts with its full burst of 3", i)
		}
	}
	ok, retryAfter := l.Allow("client-a")
	if ok {
		t.Fatal("request 4 allowed: the bucket held 3 tokens and time has not moved")
	}
	if retryAfter != time.Second {
		t.Errorf("retryAfter = %v, want 1s: at 1 token/s the next token is one second away", retryAfter)
	}
}

func TestLimiterRefillsWithTheClock(t *testing.T) {
	clk := newFakeClock()
	l := NewLimiter(1, 2, clk)

	l.Allow("client-a")
	l.Allow("client-a")
	if ok, _ := l.Allow("client-a"); ok {
		t.Fatal("bucket should be empty")
	}

	clk.Advance(2 * time.Second)
	for i := 1; i <= 2; i++ {
		if ok, _ := l.Allow("client-a"); !ok {
			t.Fatalf("request %d after 2s denied: 2 seconds at 1 token/s is 2 tokens", i)
		}
	}
	if ok, _ := l.Allow("client-a"); ok {
		t.Error("a third request was allowed: only the tokens that accrued may be spent")
	}
}

func TestLimiterCapsRefillAtBurst(t *testing.T) {
	clk := newFakeClock()
	l := NewLimiter(1, 3, clk)

	l.Allow("client-a")
	clk.Advance(time.Hour) // idle for an hour, but the bucket only holds 3

	allowed := 0
	for i := 0; i < 10; i++ {
		if ok, _ := l.Allow("client-a"); ok {
			allowed++
		}
	}
	if allowed != 3 {
		t.Errorf("allowed %d requests after an idle hour, want 3: tokens are capped at burst", allowed)
	}
}

func TestLimiterKeysAreIndependent(t *testing.T) {
	clk := newFakeClock()
	l := NewLimiter(1, 1, clk)

	if ok, _ := l.Allow("client-a"); !ok {
		t.Fatal("first request from client-a denied")
	}
	if ok, _ := l.Allow("client-a"); ok {
		t.Fatal("second request from client-a allowed")
	}
	if ok, _ := l.Allow("client-b"); !ok {
		t.Error("client-b was denied because client-a spent its own tokens: buckets are per key")
	}
}

// The server gives every request its own goroutine, so the limiter is shared
// mutable state. With a frozen clock the arithmetic is exact: 100 tokens exist
// and 500 requests compete for them, so exactly 100 may win.
func TestLimiterIsSafeForConcurrentUse(t *testing.T) {
	clk := newFakeClock()
	l := NewLimiter(1, 100, clk)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if ok, _ := l.Allow("shared"); ok {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	if allowed != 100 {
		t.Errorf("allowed %d of 500 concurrent requests, want exactly 100", allowed)
	}
}

func TestLimiterCleanupEvictsIdleBuckets(t *testing.T) {
	clk := newFakeClock()
	l := NewLimiter(1, 3, clk)

	l.Allow("gone")
	clk.Advance(time.Minute)
	l.Allow("recent")

	if removed := l.Cleanup(30 * time.Second); removed != 1 {
		t.Fatalf("Cleanup removed %d buckets, want 1 (only \"gone\" is older than 30s)", removed)
	}
	if ok, _ := l.Allow("recent"); !ok {
		t.Error("the recently used bucket was evicted")
	}
	// A bucket evicted after a full refill window is indistinguishable from a
	// new one, which is what makes eviction safe.
	if removed := l.Cleanup(30 * time.Second); removed != 0 {
		t.Errorf("second Cleanup removed %d buckets, want 0", removed)
	}
}

func TestRateLimitRefusesWith429AndRetryAfter(t *testing.T) {
	clk := newFakeClock()
	l := NewLimiter(1, 1, clk)
	mw := RateLimit(l, RemoteIP)

	served := 0
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { served++ }))

	req := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		r.RemoteAddr = "198.51.100.7:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}

	if rec := req(); rec.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", rec.Code)
	}
	rec := req()
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", rec.Code)
	}
	if served != 1 {
		t.Errorf("the handler ran %d times, want 1: a refused request must not reach it", served)
	}
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Errorf("Retry-After = %q, want %q (whole seconds, never below 1)", got, "1")
	}
	decodeEnvelope(t, rec)
}

func TestRateLimitSeparatesClientsByKey(t *testing.T) {
	clk := newFakeClock()
	l := NewLimiter(1, 1, clk)
	h := RateLimit(l, RemoteIP)(okHandler(http.StatusOK, "ok"))

	call := func(addr string) int {
		r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		r.RemoteAddr = addr
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec.Code
	}

	if got := call("203.0.113.1:1000"); got != http.StatusOK {
		t.Fatalf("first client status = %d, want 200", got)
	}
	// Same address, different source port: still the same client.
	if got := call("203.0.113.1:2000"); got != http.StatusTooManyRequests {
		t.Errorf("same IP on a new port status = %d, want 429: the port is not part of the identity", got)
	}
	if got := call("203.0.113.2:1000"); got != http.StatusOK {
		t.Errorf("second client status = %d, want 200", got)
	}
}
