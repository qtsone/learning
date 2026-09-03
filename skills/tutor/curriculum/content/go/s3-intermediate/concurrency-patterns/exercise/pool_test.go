package patterns

import (
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunPoolProcessesEveryJobOnce(t *testing.T) {
	jobs := make([]int, 40)
	for i := range jobs {
		jobs[i] = i
	}
	got := RunPool(4, jobs, func(x int) int { return x * x })
	if len(got) != len(jobs) {
		t.Fatalf("got %d results, want %d (one per job)", len(got), len(jobs))
	}
	slices.SortFunc(got, func(a, b Result) int { return a.Job - b.Job })
	for i, r := range got {
		if r.Job != i {
			t.Fatalf("job %d missing or duplicated: sorted results[%d].Job = %d", i, i, r.Job)
		}
		if r.Val != i*i {
			t.Errorf("Result for job %d has Val %d, want %d", i, r.Val, i*i)
		}
	}
}

// TestRunPoolBoundsConcurrency forces overlap with a rendezvous: every fn
// call blocks until it pairs with another concurrent call. A serial "pool"
// times out pairing; an unbounded one blows past the worker ceiling.
func TestRunPoolBoundsConcurrency(t *testing.T) {
	const workers = 3
	jobs := make([]int, 20) // even count: every rendezvous finds a partner

	var mu sync.Mutex
	inflight, maxInflight := 0, 0
	var timeouts atomic.Int32
	pair := make(chan struct{})

	fn := func(x int) int {
		mu.Lock()
		inflight++
		if inflight > maxInflight {
			maxInflight = inflight
		}
		mu.Unlock()

		select {
		case pair <- struct{}{}:
		case <-pair:
		case <-time.After(2 * time.Second):
			timeouts.Add(1)
		}

		mu.Lock()
		inflight--
		mu.Unlock()
		return x
	}

	got := RunPool(workers, jobs, fn)

	if len(got) != len(jobs) {
		t.Fatalf("got %d results, want %d", len(got), len(jobs))
	}
	if n := timeouts.Load(); n != 0 {
		t.Errorf("%d fn call(s) found no concurrent partner — the pool is not running workers in parallel", n)
	}
	mu.Lock()
	defer mu.Unlock()
	if maxInflight > workers {
		t.Errorf("saw %d concurrent fn calls, want at most %d — the pool must be bounded", maxInflight, workers)
	}
	if maxInflight < 2 {
		t.Errorf("saw at most %d concurrent fn call(s), want at least 2", maxInflight)
	}
}

func TestRunPoolDoesNotLeakGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()
	_ = RunPool(4, []int{1, 2, 3, 4, 5, 6, 7, 8}, func(x int) int { return x + 1 })

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("goroutine leak: %d goroutines before RunPool, %d well after it returned — does every worker, the feeder, and the closer exit?",
		before, runtime.NumGoroutine())
}
