package jobs

import (
	"context"
	"database/sql"
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
//
// An empty ID or Kind is an error. A zero RunAt means "due now", and a
// MaxAttempts of zero means DefaultMaxAttempts. Times are stored with
// unixNano.
func (s *Store) Enqueue(ctx context.Context, tx *sql.Tx, j NewJob) error {
	// TODO: validate, apply the defaults, and INSERT the row on tx (not on
	// s.DB — using the pool here would commit the job outside the caller's
	// transaction and hand you back the bug this design exists to remove).
	return nil
}

// Claim takes ownership of at most one job for lease long, and reports
// ErrNoJob when there is nothing to do.
//
// Two kinds of row are claimable: a ready job whose run_at has arrived, and a
// running job whose lease has expired. The second one is redelivery — the
// worker that held it crashed, and nothing else would ever notice.
//
// Claiming sets state, worker and lease_until, and counts the attempt. It must
// be a single statement: "find a job" and "mark it mine" as two round trips
// means two workers run the same job. In SQLite that is
//
//	UPDATE jobs SET ... WHERE id = (SELECT id FROM jobs WHERE ... LIMIT 1)
//	RETURNING <jobColumns>
//
// (Postgres would add FOR UPDATE SKIP LOCKED to the inner select.) Use
// scanJob to read the returned row, and errors.Is(err, sql.ErrNoRows) to
// recognise an empty queue.
func (s *Store) Claim(ctx context.Context, worker string, lease time.Duration) (Job, error) {
	// TODO: claim one job atomically.
	return Job{}, ErrNoJob
}

// Complete marks a job finished, in the caller's transaction so that it
// commits with whatever the handler wrote. Use expectOneRow so that updating a
// job that is not there is an error rather than a silent no-op.
func (s *Store) Complete(ctx context.Context, tx *sql.Tx, id string) error {
	// TODO
	return nil
}

// Retry hands a failed job back to the queue, invisible until runAt, with
// cause recorded in last_error and the lease cleared.
func (s *Store) Retry(ctx context.Context, id string, runAt time.Time, cause string) error {
	// TODO
	return nil
}

// DeadLetter takes a job out of circulation for good. Nothing retries it and
// nothing deletes it: the row is the evidence somebody has to look at.
func (s *Store) DeadLetter(ctx context.Context, id, cause string) error {
	// TODO
	return nil
}

// MarkProcessed records that this job id has had its effect applied, and
// reports whether this call is the first to do so.
//
// It runs in the handler's transaction, so "the effect happened" and "we know
// the effect happened" are one commit. A redelivery of work that already
// committed gets false here and must skip the handler.
//
// INSERT ... ON CONFLICT DO NOTHING plus RowsAffected is the whole thing.
func (s *Store) MarkProcessed(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	// TODO
	return false, nil
}
