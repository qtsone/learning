package patterns

import "context"

// Queue runs a fixed set of workers over submitted jobs and supports
// graceful shutdown: stop accepting, drain what was accepted, give up at a
// deadline. Submit is safe for concurrent use.
type Queue struct {
	// TODO: an unbuffered jobs channel, a sync.WaitGroup for the workers,
	// and a sync.Mutex guarding a closed flag — Submit must never send on a
	// closed channel (that panics).
}

// NewQueue starts `workers` goroutines that call fn for each submitted job.
// workers is at least 1.
func NewQueue(workers int, fn func(job int)) *Queue {
	// TODO: create the channel, start the workers ranging over it.
	return &Queue{}
}

// Submit hands a job to a worker, blocking until one accepts it
// (backpressure by construction). It reports whether the job was accepted:
// once Shutdown has begun, Submit returns false without running the job.
//
// TODO: under the mutex — refuse if closed, otherwise send and return true.
func (q *Queue) Submit(job int) bool {
	return false
}

// Shutdown stops intake, then waits for the workers to drain the accepted
// jobs. It returns nil on a clean drain, or ctx.Err() if ctx expires first.
// Either way, intake stays closed. Safe to call more than once.
//
// TODO: under the mutex, set the flag and close the jobs channel (once!).
// Then wrap wg.Wait in a channel and select it against ctx.Done().
func (q *Queue) Shutdown(ctx context.Context) error {
	return nil
}
