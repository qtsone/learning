package jobs

import (
	"context"
	"log/slog"
	"time"
)

// Waiter decides what an idle worker does between polls. Polling is the
// dependency-free option; production queues replace it with a notification
// (Postgres LISTEN/NOTIFY, a broker push) and keep a slow poll as the
// backstop, because a missed notification must not mean a stuck job.
type Waiter interface {
	Wait(ctx context.Context)
}

// PollEvery is a Waiter that sleeps for a fixed interval, or until the pool is
// shutting down.
type PollEvery time.Duration

// Wait sleeps for the interval unless ctx ends first.
func (d PollEvery) Wait(ctx context.Context) {
	t := time.NewTimer(time.Duration(d))
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// Pool runs jobs. It is the consumer half of the queue.
type Pool struct {
	Store    *Store
	Clock    Clock
	Handlers map[string]Handler
	Backoff  Backoff
	Waiter   Waiter

	// Workers is how many jobs may run at once. It is a concurrency limit on
	// *your* side of every dependency the handlers touch, which is usually the
	// real reason to keep it small.
	Workers int

	// Lease is how long a claim is good for. Too short and a slow-but-healthy
	// job is redelivered while it is still running; too long and a crashed
	// worker's job sits invisible for that long. Size it against the p99
	// duration of your slowest handler.
	Lease time.Duration

	Logger *slog.Logger
}

func (p *Pool) logger() *slog.Logger {
	if p.Logger != nil {
		return p.Logger
	}
	return slog.Default()
}

// ProcessOne claims at most one job and drives it to a terminal outcome:
// done, scheduled for a retry, or dead-lettered. It reports whether a job was
// claimed.
//
// Claiming honours ctx — a shutting-down pool must stop taking new work. A
// claimed job must not: run it on a context derived with
// context.WithoutCancel, so cancelling the pool never leaves a job half done
// with its lease still held.
func (p *Pool) ProcessOne(ctx context.Context, worker string) (bool, error) {
	// TODO: claim (ErrNoJob means "nothing to do", which is not a failure),
	// then run the job.
	return false, nil
}

// run executes one claimed job:
//
//   - no handler registered for job.Kind is a permanent failure — no retry
//     will deploy the missing code;
//   - open one transaction and call MarkProcessed on it. A false result is a
//     duplicate delivery: skip the handler, but still finish the job;
//   - on success, Complete in that same transaction and commit, so the
//     handler's writes and the queue update land together;
//   - on a handler error, roll back first (the effect and the processed
//     marker must not survive it) and then apply the retry policy.
func (p *Pool) run(ctx context.Context, job Job) error {
	// TODO
	return nil
}

// fail applies the retry policy to a handler error: dead-letter when the cause
// wraps ErrPermanent or the job has spent its attempt budget, otherwise Retry
// at Clock.Now() plus the backoff for this attempt.
func (p *Pool) fail(ctx context.Context, job Job, cause error) error {
	// TODO
	return nil
}

// Run starts Workers goroutines and blocks until ctx is cancelled and every
// in-flight job has reached a terminal outcome.
//
// That is the whole graceful-shutdown contract: stop claiming, finish what is
// claimed. It is politeness, not correctness — the lease is what makes a hard
// kill survivable — but it is the difference between a deploy that redelivers
// nothing and one that redelivers everything in flight.
//
// A worker loops: process one job, and when there is none, ask the Waiter and
// look again.
func (p *Pool) Run(ctx context.Context) {
	// TODO
}
