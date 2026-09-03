package board

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"
)

// Job states. A job is always in exactly one of them.
const (
	JobReady   = "ready"   // claimable once run_at has passed
	JobRunning = "running" // leased by a worker until lease_until
	JobDone    = "done"    // finished successfully; kept for auditing
	JobDead    = "dead"    // gave up; a human decides what happens next
)

// DefaultMaxAttempts is the attempt budget a job gets when the producer does
// not name one.
const DefaultMaxAttempts = 5

var (
	// ErrNoJob means nothing was claimable right now — an ordinary outcome of
	// polling an empty queue, not a failure.
	ErrNoJob = errors.New("board: no claimable job")

	// ErrPermanent marks a handler failure no retry can fix: a payload that
	// will never parse, a row that was deleted. Wrap it and the job
	// dead-letters immediately instead of burning its whole budget.
	ErrPermanent = errors.New("board: permanent failure")
)

// Job is a unit of work as stored in the queue.
type Job struct {
	ID          string
	Kind        string
	Payload     string
	State       string
	Attempts    int
	MaxAttempts int
	RunAt       time.Time
	LeaseUntil  time.Time
	Worker      string
	LastError   string
}

// NewJob is what a producer enqueues. ID is chosen by the producer because it
// is the deduplication key: it must identify the *work* ("notify:"+taskID), not
// the delivery, so that enqueueing the same work twice is a primary-key
// conflict rather than a second notification.
type NewJob struct {
	ID          string
	Kind        string
	Payload     string
	RunAt       time.Time // zero means "as soon as possible"
	MaxAttempts int       // zero means DefaultMaxAttempts
}

// JobHandler runs one job.
//
// It is handed the transaction that also marks the job finished, so a
// database-local effect and the record of that effect commit or roll back
// together. The events it returns are published by the pool *after* that
// transaction commits — announcing work that then rolls back is a lie you
// cannot take back, and a subscriber has no transaction to be rolled back with.
//
// Delivery is at-least-once and always will be, but the pool absorbs that once
// for every kind: it claims the job id in the dedup ledger on this same
// transaction and only calls Handle on the first delivery. So a handler is
// written as if it runs exactly once. Leaving that to each handler would make a
// new job kind idempotent only if its author remembered — and forgetting would
// be invisible until somebody was told something twice.
type JobHandler interface {
	Handle(ctx context.Context, tx *sql.Tx, job Job) ([]Event, error)
}

// JobHandlerFunc adapts a function to JobHandler.
type JobHandlerFunc func(ctx context.Context, tx *sql.Tx, job Job) ([]Event, error)

// Handle calls f.
func (f JobHandlerFunc) Handle(ctx context.Context, tx *sql.Tx, job Job) ([]Event, error) {
	return f(ctx, tx, job)
}

const jobColumns = `id, kind, payload, state, attempts, max_attempts, run_at, lease_until, worker, last_error`

func scanJob(sc scanner) (Job, error) {
	var (
		j                 Job
		runAt, leaseUntil int64
	)
	err := sc.Scan(&j.ID, &j.Kind, &j.Payload, &j.State, &j.Attempts, &j.MaxAttempts,
		&runAt, &leaseUntil, &j.Worker, &j.LastError)
	if err != nil {
		return Job{}, err
	}
	j.RunAt = fromUnixNano(runAt)
	j.LeaseUntil = fromUnixNano(leaseUntil)
	return j, nil
}

// Enqueue inserts a job **inside the caller's transaction**. There is no way to
// enqueue outside one, so the write that justifies the job and the job itself
// always commit together.
func (s *Store) Enqueue(ctx context.Context, tx *sql.Tx, j NewJob) error {
	if j.ID == "" || j.Kind == "" {
		return errors.New("board: enqueue needs an ID and a Kind")
	}
	runAt := j.RunAt
	if runAt.IsZero() {
		runAt = s.Clock.Now()
	}
	maxAttempts := j.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO jobs (id, kind, payload, state, attempts, max_attempts, run_at, created_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?)`,
		j.ID, j.Kind, j.Payload, JobReady, maxAttempts,
		unixNano(runAt), unixNano(s.Clock.Now()))
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", j.ID, err)
	}
	return nil
}

// Claim takes ownership of at most one job for lease long, in a single
// statement — "find a job" and "mark it mine" as two round trips is how one job
// runs twice. A running job whose lease has expired is claimable again: that is
// redelivery after a worker died, and nothing else would ever notice.
func (s *Store) Claim(ctx context.Context, worker string, lease time.Duration) (Job, error) {
	now := s.Clock.Now()
	row := s.DB.QueryRowContext(ctx, `
		UPDATE jobs
		   SET state = ?, attempts = attempts + 1, lease_until = ?, worker = ?
		 WHERE id = (
		       SELECT id FROM jobs
		        WHERE (state = ? AND run_at <= ?)
		           OR (state = ? AND lease_until <= ?)
		        ORDER BY run_at
		        LIMIT 1
		 )
		RETURNING `+jobColumns,
		JobRunning, unixNano(now.Add(lease)), worker,
		JobReady, unixNano(now),
		JobRunning, unixNano(now))

	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNoJob
	}
	if err != nil {
		return Job{}, fmt.Errorf("claim: %w", err)
	}
	return job, nil
}

// JobByID reads a job back, for tests and for operators.
func (s *Store) JobByID(ctx context.Context, id string) (Job, error) {
	job, err := scanJob(s.DB.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, fmt.Errorf("job %s: %w", id, err)
	}
	return job, nil
}

// CompleteJob marks a job finished, on the caller's transaction so that it
// commits with whatever the handler wrote.
func (s *Store) CompleteJob(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE jobs SET state = ?, lease_until = 0, worker = '' WHERE id = ?`, JobDone, id)
	if err != nil {
		return fmt.Errorf("complete %s: %w", id, err)
	}
	return nil
}

// RetryJob hands a failed job back to the queue, invisible until runAt.
func (s *Store) RetryJob(ctx context.Context, id string, runAt time.Time, cause string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE jobs SET state = ?, run_at = ?, lease_until = 0, worker = '', last_error = ?
		 WHERE id = ?`,
		JobReady, unixNano(runAt), cause, id)
	if err != nil {
		return fmt.Errorf("retry %s: %w", id, err)
	}
	return nil
}

// DeadLetterJob takes a job out of circulation for good. Nothing retries it and
// nothing deletes it: the row, with its last_error, is the evidence somebody
// reads at 09:00.
func (s *Store) DeadLetterJob(ctx context.Context, id, cause string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE jobs SET state = ?, lease_until = 0, worker = '', last_error = ?
		 WHERE id = ?`,
		JobDead, cause, id)
	if err != nil {
		return fmt.Errorf("dead-letter %s: %w", id, err)
	}
	return nil
}

// MarkProcessed records that this job id has had its effect applied, and
// reports whether this call is the first to do so.
//
// The pool calls it on the transaction it then hands the handler, so "the effect
// happened" and "we know it happened" are one commit — and a failure rolls the
// marker back along with the effect, so the retry is a real retry and not a
// skipped one.
func (s *Store) MarkProcessed(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	res, err := tx.ExecContext(ctx, `
		INSERT INTO processed_jobs (job_id, processed_at) VALUES (?, ?)
		ON CONFLICT (job_id) DO NOTHING`,
		id, unixNano(s.Clock.Now()))
	if err != nil {
		return false, fmt.Errorf("mark processed %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("mark processed %s: %w", id, err)
	}
	return n == 1, nil
}

// Backoff is the retry schedule: exponential from Base, capped at Max, with
// equal jitter so a thousand jobs failed by one outage do not retry in lockstep
// and become the next outage.
type Backoff struct {
	Base time.Duration
	Max  time.Duration

	// Rand returns a value in [0,n). Injected for the same reason the clock is:
	// a test asserts the policy instead of sampling it. Nil means math/rand/v2.
	Rand func(n int64) int64
}

// Delay reports how long to wait before attempt+1.
func (b Backoff) Delay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := b.Base
	for i := 1; i < attempt && d < b.Max; i++ {
		d *= 2
	}
	if d > b.Max {
		d = b.Max
	}
	half := int64(d / 2)
	if half <= 0 {
		return d
	}
	r := b.Rand
	if r == nil {
		r = rand.Int64N
	}
	return time.Duration(half + r(half))
}

// Pool is the consumer half of the queue: it claims jobs, runs them, and
// applies the retry policy.
type Pool struct {
	Store    *Store
	Clock    Clock
	Hub      *Hub
	Handlers map[string]JobHandler
	Backoff  Backoff

	// Workers is how many jobs may run at once — a concurrency limit on *your*
	// side of every dependency the handlers touch.
	Workers int

	// Lease is how long a claim is good for. Too short and a slow-but-healthy
	// job is redelivered while it is still running; too long and a crashed
	// worker's job sits invisible for that long.
	Lease time.Duration

	// Idle is how long a worker waits before polling an empty queue again.
	Idle time.Duration

	Logger *slog.Logger
}

func (p *Pool) logger() *slog.Logger {
	if p.Logger != nil {
		return p.Logger
	}
	return slog.Default()
}

// ProcessOne claims at most one job and drives it to exactly one terminal
// outcome: done, scheduled for a retry, or dead-lettered. It reports whether a
// job was claimed.
//
// Claiming honours ctx — a shutting-down pool stops taking new work — while the
// claimed job runs on a context derived with context.WithoutCancel, so a
// cancelled pool never leaves a job half done with its lease still held.
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

func (p *Pool) run(ctx context.Context, job Job) error {
	h, ok := p.Handlers[job.Kind]
	if !ok {
		// No retry will conjure a handler. This is a deploy problem, and the
		// dead-letter row is how somebody finds out.
		return p.Store.DeadLetterJob(ctx, job.ID, "no handler registered for kind "+job.Kind)
	}

	tx, err := p.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("run %s: %w", job.ID, err)
	}
	defer tx.Rollback()

	// The dedup gate lives here rather than in the handlers so that no kind of
	// job can forget it, and so that it covers the events as well as the effect:
	// a handler that never runs returns nothing for the pool to publish.
	first, err := p.Store.MarkProcessed(ctx, tx, job.ID)
	if err != nil {
		return err
	}

	var events []Event
	if first {
		var herr error
		events, herr = h.Handle(ctx, tx, job)
		if herr != nil {
			// Roll back before touching the queue: the effect must not survive a
			// failure, and neither must the marker, or the retry would skip the
			// work it is retrying.
			_ = tx.Rollback()
			return p.fail(ctx, job, herr)
		}
	} else {
		p.logger().Info("duplicate delivery skipped", "job", job.ID, "attempt", job.Attempts)
	}

	if err := p.Store.CompleteJob(ctx, tx, job.ID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("run %s: %w", job.ID, err)
	}

	// Only now. Everything above can still be undone; a delivered event cannot.
	for _, e := range events {
		p.Hub.Publish(e)
	}
	return nil
}

func (p *Pool) fail(ctx context.Context, job Job, cause error) error {
	if errors.Is(cause, ErrPermanent) || job.Attempts >= job.MaxAttempts {
		p.logger().Error("job dead-lettered",
			"job", job.ID, "kind", job.Kind, "attempts", job.Attempts, "err", cause)
		return p.Store.DeadLetterJob(ctx, job.ID, cause.Error())
	}
	delay := p.Backoff.Delay(job.Attempts)
	p.logger().Warn("job failed, retrying",
		"job", job.ID, "kind", job.Kind, "attempts", job.Attempts, "in", delay, "err", cause)
	return p.Store.RetryJob(ctx, job.ID, p.Clock.Now().Add(delay), cause.Error())
}

// Run starts Workers goroutines and blocks until ctx is cancelled *and* every
// in-flight job has reached a terminal outcome: stop claiming, finish what is
// claimed. That is politeness — the lease is what makes a hard kill survivable.
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
		timer := time.NewTimer(p.Idle)
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
		timer.Stop()
	}
}
