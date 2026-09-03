package patterns

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestQueueDrainsEverythingOnShutdown(t *testing.T) {
	var mu sync.Mutex
	seen := make(map[int]int)
	q := NewQueue(3, func(job int) {
		mu.Lock()
		seen[job]++
		mu.Unlock()
	})

	const jobs = 20
	for i := range jobs {
		if !q.Submit(i) {
			t.Fatalf("Submit(%d) = false before shutdown, want true", i)
		}
	}
	if err := q.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown with no deadline = %v, want nil after a clean drain", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != jobs {
		t.Errorf("%d distinct jobs processed, want %d — Shutdown must drain accepted work before returning", len(seen), jobs)
	}
	for job, n := range seen {
		if n != 1 {
			t.Errorf("job %d processed %d times, want exactly once", job, n)
		}
	}
}

func TestQueueRefusesWorkAfterShutdown(t *testing.T) {
	var mu sync.Mutex
	seen := make(map[int]bool)
	q := NewQueue(2, func(job int) {
		mu.Lock()
		seen[job] = true
		mu.Unlock()
	})
	if !q.Submit(1) {
		t.Fatal("Submit(1) = false before shutdown, want true")
	}
	if err := q.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown = %v, want nil", err)
	}
	if q.Submit(2) {
		t.Error("Submit(2) = true after Shutdown, want false — intake must be closed")
	}
	mu.Lock()
	defer mu.Unlock()
	if seen[2] {
		t.Error("job 2 ran even though it was submitted after Shutdown")
	}
}

func TestQueueShutdownHonorsDeadline(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	q := NewQueue(1, func(job int) {
		entered <- struct{}{}
		<-release // this job is stuck until the test frees it
	})
	if !q.Submit(1) {
		t.Fatal("Submit(1) = false, want true")
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the worker never picked up the job — does NewQueue start workers ranging over the jobs channel?")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // deadline already expired: draining can't possibly finish
	if err := q.Shutdown(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Shutdown with expired context = %v, want context.Canceled — a stuck job must not hang Shutdown past its deadline", err)
	}
	if q.Submit(2) {
		t.Error("Submit(2) = true after a timed-out Shutdown, want false — intake stays closed even when draining gave up")
	}
	close(release) // free the stuck worker so nothing outlives the test
}
