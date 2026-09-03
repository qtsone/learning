package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// fakeClock is the whole reason this suite has no sleeps in it. Lease expiry,
// delayed jobs and retry backoff are all "compare a stored timestamp with
// Clock.Now()", so a clock the test advances by hand makes every one of those
// rules deterministic under -race.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestStore(t *testing.T) (*Store, *fakeClock) {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "jobs.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	clock := newFakeClock()
	return NewStore(db, clock), clock
}

// quietLogger keeps expected failures out of the test output.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestPool(t *testing.T, store *Store, clock Clock, h Handler) *Pool {
	t.Helper()
	return &Pool{
		Store:    store,
		Clock:    clock,
		Handlers: map[string]Handler{"send-confirmation": h},
		Backoff:  Backoff{Base: time.Second, Max: time.Minute, Rand: func(int64) int64 { return 0 }},
		Waiter:   blockingWaiter{},
		Workers:  1,
		Lease:    30 * time.Second,
		Logger:   quietLogger(),
	}
}

// blockingWaiter parks an idle worker until the pool shuts down. Tests enqueue
// everything before starting the pool, so there is nothing to poll for.
type blockingWaiter struct{}

func (blockingWaiter) Wait(ctx context.Context) { <-ctx.Done() }

// recorder is the job handler: it sends the confirmation e-mail for an order
// by writing a row, inside the transaction it is handed. It also counts its
// calls, which is how the suite asserts "this effect happened exactly once"
// without measuring anything.
type recorder struct {
	clock Clock

	mu      sync.Mutex
	calls   int
	failFn  func(call int) error
	block   chan struct{}
	started chan struct{}
}

func (h *recorder) Handle(ctx context.Context, tx *sql.Tx, job Job) error {
	h.mu.Lock()
	h.calls++
	call := h.calls
	failFn, block, started := h.failFn, h.block, h.started
	h.mu.Unlock()

	if started != nil {
		started <- struct{}{}
	}
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if failFn != nil {
		if err := failFn(call); err != nil {
			return err
		}
	}

	var address string
	err := tx.QueryRowContext(ctx, `SELECT email FROM orders WHERE id = ?`, job.Payload).Scan(&address)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: no order %q", ErrPermanent, job.Payload)
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO emails (order_id, address, sent_at) VALUES (?, ?, ?)`,
		job.Payload, address, unixNano(h.clock.Now()))
	return err
}

func (h *recorder) Calls() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.calls
}

func seedOrder(t *testing.T, store *Store, id, email string) {
	t.Helper()
	_, err := store.DB.Exec(`INSERT INTO orders (id, email, total_cents) VALUES (?, ?, 1000)`, id, email)
	if err != nil {
		t.Fatalf("seed order %s: %v", id, err)
	}
}

// mustEnqueue commits one job through the same transaction-only door
// producers use.
func mustEnqueue(t *testing.T, store *Store, j NewJob) {
	t.Helper()
	ctx := context.Background()
	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	if err := store.Enqueue(ctx, tx, j); err != nil {
		t.Fatalf("Enqueue(%s): unexpected error: %v", j.ID, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func mustGet(t *testing.T, store *Store, id string) Job {
	t.Helper()
	job, err := store.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get(%s): %v", id, err)
	}
	return job
}

// waitFor receives one value from ch, or fails instead of hanging the suite
// until the package timeout. It is a liveness guard, not a timing assertion:
// nothing in this lesson's contract depends on wall-clock time.
func waitFor[T any](t *testing.T, ch <-chan T, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}
