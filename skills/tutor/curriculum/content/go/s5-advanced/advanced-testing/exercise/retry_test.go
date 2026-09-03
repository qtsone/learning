package logkit

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
)

// fakeClock is a stub: it answers "the sleep happened" without any real time
// passing, and records what it was asked to wait for. Ten lines, no
// dependencies, and the test can assert on the exact backoff sequence — which
// is the only honest way to test timing under the race detector.
type fakeClock struct {
	slept []time.Duration
	err   error // when non-nil, Sleep reports this instead of sleeping
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	c.slept = append(c.slept, d)
	if c.err != nil {
		return c.err
	}
	return ctx.Err()
}

var errBoom = errors.New("boom")

func TestRetry(t *testing.T) {
	policy := Policy{MaxAttempts: 4, Base: 100 * time.Millisecond, Max: 250 * time.Millisecond}

	cases := []struct {
		name      string
		failures  int // how many leading attempts fail with errBoom
		wantCalls int
		wantSlept []time.Duration
		wantErr   error // nil means success
	}{
		{"succeeds immediately", 0, 1, nil, nil},
		{"succeeds on the third attempt", 2, 3, []time.Duration{100 * time.Millisecond, 200 * time.Millisecond}, nil},
		{
			name:      "exhausts the budget",
			failures:  99,
			wantCalls: 4,
			wantSlept: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 250 * time.Millisecond},
			wantErr:   errBoom,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clk := &fakeClock{}
			calls := 0
			err := Retry(context.Background(), clk, policy, func(context.Context) error {
				calls++
				if calls <= c.failures {
					return errBoom
				}
				return nil
			})
			if calls != c.wantCalls {
				t.Errorf("fn called %d times, want %d", calls, c.wantCalls)
			}
			if !slices.Equal(clk.slept, c.wantSlept) {
				t.Errorf("slept %v, want %v (no sleep after the final attempt)", clk.slept, c.wantSlept)
			}
			if c.wantErr == nil {
				if err != nil {
					t.Errorf("Retry() = %v, want nil", err)
				}
			} else if !errors.Is(err, c.wantErr) {
				t.Errorf("Retry() = %v, want an error matching %v", err, c.wantErr)
			}
		})
	}
}

func TestRetryStopsOnPermanentError(t *testing.T) {
	clk := &fakeClock{}
	calls := 0
	err := Retry(context.Background(), clk, Policy{MaxAttempts: 5, Base: time.Second, Max: time.Minute},
		func(context.Context) error {
			calls++
			if calls == 1 {
				return errBoom
			}
			return fmt.Errorf("bad credentials: %w", ErrPermanent)
		})
	if calls != 2 {
		t.Errorf("fn called %d times, want 2 (stop as soon as the failure is permanent)", calls)
	}
	if len(clk.slept) != 1 {
		t.Errorf("slept %v, want exactly one delay", clk.slept)
	}
	if !errors.Is(err, ErrPermanent) {
		t.Errorf("Retry() = %v, want an error matching ErrPermanent", err)
	}
}

func TestRetryHonoursContext(t *testing.T) {
	t.Run("canceled before the first attempt", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		clk := &fakeClock{}
		calls := 0
		err := Retry(ctx, clk, Policy{MaxAttempts: 3, Base: time.Second, Max: time.Minute},
			func(context.Context) error { calls++; return errBoom })
		if calls != 0 {
			t.Errorf("fn called %d times, want 0 on an already-canceled context", calls)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Retry() = %v, want an error matching context.Canceled", err)
		}
	})

	t.Run("canceled during a backoff sleep", func(t *testing.T) {
		clk := &fakeClock{err: context.Canceled}
		calls := 0
		err := Retry(context.Background(), clk, Policy{MaxAttempts: 3, Base: time.Second, Max: time.Minute},
			func(context.Context) error { calls++; return errBoom })
		if calls != 1 {
			t.Errorf("fn called %d times, want 1 (the sleep was cut short)", calls)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Retry() = %v, want an error matching context.Canceled", err)
		}
		if !errors.Is(err, errBoom) {
			t.Errorf("Retry() = %v, want it to also carry the last attempt's error", err)
		}
	})
}

func TestRetryRejectsImpossiblePolicy(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), &fakeClock{}, Policy{MaxAttempts: 0, Base: time.Second, Max: time.Minute},
		func(context.Context) error { calls++; return nil })
	if err == nil {
		t.Error("Retry() = nil for MaxAttempts 0, want an error")
	}
	if calls != 0 {
		t.Errorf("fn called %d times, want 0", calls)
	}
}

func TestBackoffTable(t *testing.T) {
	p := Policy{MaxAttempts: 10, Base: 100 * time.Millisecond, Max: 800 * time.Millisecond}
	want := []time.Duration{100, 200, 400, 800, 800, 800}
	for i, ms := range want {
		attempt := i + 1
		if got := p.Backoff(attempt); got != ms*time.Millisecond {
			t.Errorf("Backoff(%d) = %v, want %v", attempt, got, ms*time.Millisecond)
		}
	}
}

// A property test: instead of listing outputs, state what must hold for every
// input and let the loop check a wide range of them. This is the shape you
// scale up with fuzzing.
func TestBackoffProperties(t *testing.T) {
	p := Policy{MaxAttempts: 64, Base: time.Millisecond, Max: 250 * time.Millisecond}
	prev := time.Duration(0)
	for attempt := 1; attempt <= 64; attempt++ {
		d := p.Backoff(attempt)
		if d < p.Base || d > p.Max {
			t.Fatalf("Backoff(%d) = %v, want between %v and %v", attempt, d, p.Base, p.Max)
		}
		if d < prev {
			t.Fatalf("Backoff(%d) = %v, less than Backoff(%d) = %v; backoff must never shrink", attempt, d, attempt-1, prev)
		}
		prev = d
	}
}

func TestRealClockSleep(t *testing.T) {
	var clk Clock = RealClock{}
	if err := clk.Sleep(context.Background(), time.Millisecond); err != nil {
		t.Errorf("Sleep(1ms) = %v, want nil", err)
	}

	// If Sleep ignores the context, this test takes 30 seconds instead of no
	// time at all — which is exactly the bug it exists to catch.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := clk.Sleep(ctx, 30*time.Second); !errors.Is(err, context.Canceled) {
		t.Errorf("Sleep on a canceled context = %v, want context.Canceled", err)
	}
}
