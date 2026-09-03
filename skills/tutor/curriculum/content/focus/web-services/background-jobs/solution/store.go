package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Store is the queue. It owns no goroutines and no timers: every time it needs
// "now" it asks the injected Clock, which is why the tests can make a lease
// expire without waiting for one.
type Store struct {
	DB    *sql.DB
	Clock Clock
}

// NewStore builds a Store over an already-open database.
func NewStore(db *sql.DB, clock Clock) *Store {
	return &Store{DB: db, Clock: clock}
}

const jobColumns = `id, kind, payload, state, attempts, max_attempts, run_at, lease_until, worker, last_error`

// Enqueue inserts a job **inside the caller's transaction**.
//
// The signature is the lesson: there is no way to enqueue outside a
// transaction, so the write that justifies the job and the job itself always
// commit together. A producer with nothing else to write opens a transaction
// of one statement and pays almost nothing for the habit.
func (s *Store) Enqueue(ctx context.Context, tx *sql.Tx, j NewJob) error {
	if j.ID == "" || j.Kind == "" {
		return errors.New("jobs: enqueue needs an ID and a Kind")
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
		j.ID, j.Kind, j.Payload, StateReady, maxAttempts,
		unixNano(runAt), unixNano(s.Clock.Now()))
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", j.ID, err)
	}
	return nil
}

// Claim takes ownership of at most one job for lease long, and reports
// ErrNoJob when there is nothing to do.
//
// One statement does the whole thing, because "find a job" and "mark it mine"
// must not be two round trips: between them another worker claims the same
// row and the job runs twice. Two kinds of row are claimable — a ready job
// whose run_at has arrived, and a running job whose lease has expired. The
// second is redelivery: the worker that held it crashed, and nothing else
// would ever notice.
//
// attempts is incremented here, at claim time, not when a handler returns an
// error. A job that kills its worker never returns anything, and counting only
// clean failures would let it be redelivered forever.
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
		StateRunning, unixNano(now.Add(lease)), worker,
		StateReady, unixNano(now),
		StateRunning, unixNano(now))

	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNoJob
	}
	if err != nil {
		return Job{}, fmt.Errorf("claim: %w", err)
	}
	return job, nil
}

// Complete marks a job finished, in the caller's transaction so that it
// commits with whatever the handler wrote.
func (s *Store) Complete(ctx context.Context, tx *sql.Tx, id string) error {
	res, err := tx.ExecContext(ctx,
		`UPDATE jobs SET state = ?, lease_until = 0, worker = '' WHERE id = ?`,
		StateDone, id)
	if err != nil {
		return fmt.Errorf("complete %s: %w", id, err)
	}
	return expectOneRow(res, "complete", id)
}

// Retry hands a failed job back to the queue, invisible until runAt.
func (s *Store) Retry(ctx context.Context, id string, runAt time.Time, cause string) error {
	res, err := s.DB.ExecContext(ctx, `
		UPDATE jobs
		   SET state = ?, run_at = ?, lease_until = 0, worker = '', last_error = ?
		 WHERE id = ?`,
		StateReady, unixNano(runAt), cause, id)
	if err != nil {
		return fmt.Errorf("retry %s: %w", id, err)
	}
	return expectOneRow(res, "retry", id)
}

// DeadLetter takes a job out of circulation for good. Nothing retries it and
// nothing deletes it: the row is the evidence somebody has to look at.
func (s *Store) DeadLetter(ctx context.Context, id, cause string) error {
	res, err := s.DB.ExecContext(ctx, `
		UPDATE jobs
		   SET state = ?, lease_until = 0, worker = '', last_error = ?
		 WHERE id = ?`,
		StateDead, cause, id)
	if err != nil {
		return fmt.Errorf("dead-letter %s: %w", id, err)
	}
	return expectOneRow(res, "dead-letter", id)
}

// MarkProcessed records that this job id has had its effect applied, and
// reports whether this call is the first to do so.
//
// It runs in the handler's transaction, so "the effect happened" and "we know
// the effect happened" are one commit. A redelivery of work that already
// committed gets false here and must skip the handler.
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
