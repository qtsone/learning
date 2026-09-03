package patterns

import (
	"context"
	"sync"
)

// Queue runs a fixed set of workers over submitted jobs and supports
// graceful shutdown: stop accepting, drain what was accepted, give up at a
// deadline. Submit is safe for concurrent use.
type Queue struct {
	jobs   chan int
	wg     sync.WaitGroup
	mu     sync.Mutex
	closed bool
}

// NewQueue starts `workers` goroutines that call fn for each submitted job.
// workers is at least 1.
func NewQueue(workers int, fn func(job int)) *Queue {
	q := &Queue{jobs: make(chan int)}
	q.wg.Add(workers)
	for range workers {
		go func() {
			defer q.wg.Done()
			for job := range q.jobs {
				fn(job)
			}
		}()
	}
	return q
}

// Submit hands a job to a worker, blocking until one accepts it
// (backpressure by construction). It reports whether the job was accepted:
// once Shutdown has begun, Submit returns false without running the job.
func (q *Queue) Submit(job int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return false
	}
	// Sending under the mutex is what makes close(q.jobs) safe: Shutdown
	// takes the same mutex, so no Submit can be mid-send when it closes.
	q.jobs <- job
	return true
}

// Shutdown stops intake, then waits for the workers to drain the accepted
// jobs. It returns nil on a clean drain, or ctx.Err() if ctx expires first.
// Either way, intake stays closed. Safe to call more than once.
//
// The deadline guards slow drains, not a wedged fn: if every worker is
// permanently stuck in fn while a Submit is blocked mid-handoff, that
// Submit holds the mutex and Shutdown blocks acquiring it before it ever
// reaches the deadline. See TUTOR.md — naming this envelope is part of
// defending the design.
func (q *Queue) Shutdown(ctx context.Context) error {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		close(q.jobs)
	}
	q.mu.Unlock()

	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
