package jobs

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Job states. A job is always in exactly one of them.
const (
	StateReady   = "ready"   // claimable once run_at has passed
	StateRunning = "running" // leased by a worker until lease_until
	StateDone    = "done"    // finished successfully; kept for auditing
	StateDead    = "dead"    // gave up; a human decides what happens next
)

// DefaultMaxAttempts is the attempt budget a job gets when the producer does
// not name one. "Retry forever" is not a policy: it turns one poison message
// into an infinite loop that starves every other job.
const DefaultMaxAttempts = 5

var (
	// ErrNoJob means nothing was claimable right now. It is an ordinary
	// outcome of polling an empty queue, not a failure.
	ErrNoJob = errors.New("jobs: no claimable job")

	// ErrPermanent marks a handler failure that no retry can fix: a malformed
	// payload, a deleted record, a 400 from a downstream API. Wrap it
	// (fmt.Errorf("%w: ...", ErrPermanent)) and the job dead-letters
	// immediately instead of burning its whole attempt budget.
	ErrPermanent = errors.New("jobs: permanent failure")
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

// NewJob is what a producer enqueues.
//
// ID is chosen by the producer on purpose. It is the deduplication key the
// consumer keys on, so it must identify the *work*, not the delivery: a
// natural id like "order-confirmation:"+orderID makes a second enqueue of the
// same work a primary-key conflict instead of a second e-mail.
type NewJob struct {
	ID          string
	Kind        string
	Payload     string
	RunAt       time.Time // zero means "as soon as possible"
	MaxAttempts int       // zero means DefaultMaxAttempts
}

// Handler runs one job.
//
// It is handed the *same transaction* that marks the job processed and
// finished, so a database-local effect and the record of that effect commit or
// roll back together — there is no window in which the effect happened but the
// queue does not know. An effect that cannot join this transaction (an HTTP
// call to a payment provider, a file written to disk) does not get that
// guarantee and must carry its own idempotency key.
type Handler interface {
	Handle(ctx context.Context, tx *sql.Tx, job Job) error
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(ctx context.Context, tx *sql.Tx, job Job) error

// Handle calls f.
func (f HandlerFunc) Handle(ctx context.Context, tx *sql.Tx, job Job) error {
	return f(ctx, tx, job)
}

// Order is the business record this service exists for. The confirmation
// e-mail it triggers is the job.
type Order struct {
	ID         string
	Email      string
	TotalCents int
}
