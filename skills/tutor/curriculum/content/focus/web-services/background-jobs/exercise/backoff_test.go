package jobs

import (
	"testing"
	"time"
)

func TestBackoffDoublesAndCaps(t *testing.T) {
	// Rand always returns 0, so every delay is the low end of its jitter
	// window: exactly half the nominal delay for that attempt.
	b := Backoff{Base: time.Second, Max: 30 * time.Second, Rand: func(int64) int64 { return 0 }}

	tests := []struct {
		attempt int
		nominal time.Duration
	}{
		{attempt: 0, nominal: time.Second}, // an attempt below 1 is treated as the first
		{attempt: 1, nominal: time.Second},
		{attempt: 2, nominal: 2 * time.Second},
		{attempt: 3, nominal: 4 * time.Second},
		{attempt: 4, nominal: 8 * time.Second},
		{attempt: 6, nominal: 30 * time.Second}, // 32s, capped at Max
		{attempt: 40, nominal: 30 * time.Second},
	}
	for _, tc := range tests {
		got := b.Delay(tc.attempt)
		want := tc.nominal / 2
		if got != want {
			t.Errorf("Delay(%d) = %v, want %v (half of the nominal %v, with jitter at its minimum)",
				tc.attempt, got, want, tc.nominal)
		}
	}
}

func TestBackoffJitterStaysInsideItsWindow(t *testing.T) {
	// The real source this time: the contract is a range, not a value.
	b := Backoff{Base: time.Second, Max: time.Minute}

	seen := map[time.Duration]bool{}
	for i := 0; i < 200; i++ {
		got := b.Delay(3) // nominal 4s
		if got < 2*time.Second || got > 4*time.Second {
			t.Fatalf("Delay(3) = %v, want a value in [2s, 4s]", got)
		}
		seen[got] = true
	}
	if len(seen) < 2 {
		t.Errorf("Delay(3) returned the same value %d times: without jitter, every job that "+
			"failed together retries together and the stampede repeats at every boundary", 200)
	}
}

func TestBackoffUsesTheInjectedSource(t *testing.T) {
	var asked int64
	b := Backoff{
		Base: 4 * time.Second,
		Max:  time.Minute,
		Rand: func(n int64) int64 { asked = n; return n - 1 },
	}

	got := b.Delay(1)
	if want := 4 * time.Second; got != want {
		t.Errorf("Delay(1) with the highest jitter = %v, want %v (the full nominal delay)", got, want)
	}
	if want := int64(2*time.Second) + 1; asked != want {
		t.Errorf("Rand was asked for a value in [0,%d), want [0,%d): half the delay is fixed "+
			"and half is jittered", asked, want)
	}
}
