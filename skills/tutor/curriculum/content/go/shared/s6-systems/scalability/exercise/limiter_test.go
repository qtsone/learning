package scalability

import (
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}
func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestBurstUpToCapacityThenDeny(t *testing.T) {
	clk := newFakeClock()
	tb := NewTokenBucket(5, 1, clk.now)
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Fatalf("request %d of a burst of 5 denied; bucket starts full with capacity 5", i+1)
		}
	}
	if tb.Allow() {
		t.Fatal("request 6 allowed; bucket of capacity 5 must be empty after a burst of 5")
	}
}

func TestRefillOverTime(t *testing.T) {
	clk := newFakeClock()
	tb := NewTokenBucket(2, 1, clk.now) // 1 token per second
	tb.Allow()
	tb.Allow()

	clk.advance(500 * time.Millisecond)
	if tb.Allow() {
		t.Fatal("allowed after 500ms at 1 token/s: half a token is not a token")
	}
	clk.advance(500 * time.Millisecond)
	if !tb.Allow() {
		t.Fatal("denied after a full second at 1 token/s: one token should have refilled")
	}
	if tb.Allow() {
		t.Fatal("second request allowed after 1s at 1 token/s: only one token refilled")
	}
}

func TestRefillNeverExceedsCapacity(t *testing.T) {
	clk := newFakeClock()
	tb := NewTokenBucket(3, 100, clk.now)
	for i := 0; i < 3; i++ {
		tb.Allow()
	}
	clk.advance(time.Hour) // enough for 360,000 tokens — bucket holds 3
	for i := 0; i < 3; i++ {
		if !tb.Allow() {
			t.Fatalf("request %d denied after a long idle: bucket should be full again", i+1)
		}
	}
	if tb.Allow() {
		t.Fatal("request 4 allowed: refill must cap at capacity, not accumulate forever")
	}
}

func TestRetryAfter(t *testing.T) {
	clk := newFakeClock()
	tb := NewTokenBucket(1, 2, clk.now) // 2 tokens per second

	if got := tb.RetryAfter(); got != 0 {
		t.Fatalf("RetryAfter() with a token available = %v; want 0", got)
	}
	tb.Allow()
	if got := tb.RetryAfter(); got != 500*time.Millisecond {
		t.Fatalf("RetryAfter() on an empty bucket at 2 tokens/s = %v; want 500ms", got)
	}
}

func TestWaitingRetryAfterSucceeds(t *testing.T) {
	clk := newFakeClock()
	tb := NewTokenBucket(4, 3, clk.now)
	for i := 0; i < 4; i++ {
		tb.Allow()
	}
	clk.advance(123 * time.Millisecond) // land on an awkward partial refill
	tb.Allow()

	wait := tb.RetryAfter()
	if wait <= 0 {
		t.Fatalf("RetryAfter() on an empty bucket = %v; want > 0", wait)
	}
	clk.advance(wait + time.Millisecond)
	if !tb.Allow() {
		t.Fatalf("denied after waiting the advertised RetryAfter (%v): the promise a 429 makes must hold", wait)
	}
}
