package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
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
// claimed job does not: it runs on a context derived with
// context.WithoutCancel, so cancelling the pool never leaves a job half done
// with its lease still held.
func (p *Pool) ProcessOne(ctx context.Context, worker string) (bool, error) {
	job, err := p.Store.Claim(ctx, worker, p.Lease)
	if errors.Is(err, ErrNoJob) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, p.run(context.WithoutCancel(ctx), job)
}

// run executes one claimed job.
func (p *Pool) run(ctx context.Context, job Job) error {
	h, ok := p.Handlers[job.Kind]
	if !ok {
		// No retry will conjure a handler. This is a deploy problem, and the
		// dead-letter row is how somebody finds out.
		return p.Store.DeadLetter(ctx, job.ID, "no handler registered for kind "+job.Kind)
	}

	tx, err := p.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("run %s: %w", job.ID, err)
	}
	defer tx.Rollback()

	first, err := p.Store.MarkProcessed(ctx, tx, job.ID)
	if err != nil {
		return err
	}
	if first {
		if herr := h.Handle(ctx, tx, job); herr != nil {
			// Roll back before touching the queue: the effect must not
			// survive, and neither must the processed marker, or the retry
			// would skip the work it is retrying.
			_ = tx.Rollback()
			return p.fail(ctx, job, herr)
		}
	} else {
		p.logger().Info("duplicate delivery skipped", "job", job.ID, "attempt", job.Attempts)
	}

	if err := p.Store.Complete(ctx, tx, job.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// fail applies the retry policy to a handler error.
func (p *Pool) fail(ctx context.Context, job Job, cause error) error {
	if errors.Is(cause, ErrPermanent) || job.Attempts >= job.MaxAttempts {
		p.logger().Error("job dead-lettered",
			"job", job.ID, "kind", job.Kind, "attempts", job.Attempts, "err", cause)
		return p.Store.DeadLetter(ctx, job.ID, cause.Error())
	}
	delay := p.Backoff.Delay(job.Attempts)
	p.logger().Warn("job failed, retrying",
		"job", job.ID, "kind", job.Kind, "attempts", job.Attempts, "in", delay, "err", cause)
	return p.Store.Retry(ctx, job.ID, p.Clock.Now().Add(delay), cause.Error())
}

// Run starts Workers goroutines and blocks until ctx is cancelled and every
// in-flight job has reached a terminal outcome.
//
// That is the whole graceful-shutdown contract: stop claiming, finish what is
// claimed. It is politeness, not correctness — the lease is what makes a hard
// kill survivable — but it is the difference between a deploy that redelivers
// nothing and one that redelivers everything in flight.
func (p *Pool) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < p.Workers; i++ {
		id := fmt.Sprintf("worker-%d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.work(ctx, id)
		}()
	}
	wg.Wait()
}

func (p *Pool) work(ctx context.Context, id string) {
	for ctx.Err() == nil {
		claimed, err := p.ProcessOne(ctx, id)
		switch {
		case err != nil:
			if ctx.Err() != nil {
				return
			}
			p.logger().Error("worker failed", "worker", id, "err", err)
		case claimed:
			continue
		}
		p.Waiter.Wait(ctx)
	}
}
