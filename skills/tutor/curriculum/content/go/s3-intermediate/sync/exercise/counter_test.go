package syncex

import (
	"sync"
	"testing"
)

func TestCounterSequential(t *testing.T) {
	c := NewCounter()
	c.Inc("get")
	c.Inc("get")
	c.Inc("post")
	cases := []struct {
		name string
		want int
	}{
		{"get", 2},
		{"post", 1},
		{"delete", 0},
	}
	for _, tc := range cases {
		if got := c.Value(tc.name); got != tc.want {
			t.Errorf("Value(%q) = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestCounterConcurrent(t *testing.T) {
	c := NewCounter()
	const goroutines, perGoroutine = 8, 200
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				c.Inc("requests")
			}
		}()
	}
	wg.Wait()
	want := goroutines * perGoroutine
	if got := c.Value("requests"); got != want {
		t.Errorf("after %d goroutines x %d Incs: Value(\"requests\") = %d, want %d (is every map access inside the lock?)",
			goroutines, perGoroutine, got, want)
	}
}

func TestCounterReadsDuringWrites(t *testing.T) {
	c := NewCounter()
	const n = 200
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range n {
			c.Inc("x")
		}
	}()
	go func() {
		defer wg.Done()
		for range n {
			_ = c.Value("x")
			_ = c.Snapshot()
		}
	}()
	wg.Wait()
	if got := c.Value("x"); got != n {
		t.Errorf("Value(\"x\") = %d, want %d — reads and writes must share one lock", got, n)
	}
}

func TestCounterSnapshotIsACopy(t *testing.T) {
	c := NewCounter()
	c.Inc("a")
	snap := c.Snapshot()
	if snap == nil {
		t.Fatal("Snapshot() = nil, want a copy of the counts")
	}
	if got := snap["a"]; got != 1 {
		t.Fatalf("Snapshot()[\"a\"] = %d, want 1", got)
	}
	snap["a"] = 100
	if got := c.Value("a"); got != 1 {
		t.Errorf("mutating the snapshot changed the Counter: Value(\"a\") = %d, want 1 (return a copy, not the live map)", got)
	}
}

func TestHitsConcurrent(t *testing.T) {
	var h Hits
	const goroutines, perGoroutine = 8, 200
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				h.Inc()
			}
		}()
	}
	wg.Wait()
	if got, want := h.Value(), int64(goroutines*perGoroutine); got != want {
		t.Errorf("after %d goroutines x %d Incs: Value() = %d, want %d", goroutines, perGoroutine, got, want)
	}
}
