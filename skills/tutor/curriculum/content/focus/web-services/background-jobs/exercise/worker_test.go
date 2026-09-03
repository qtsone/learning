package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var errSMTP = errors.New("smtp: connection reset")

func TestProcessOneRunsTheJobAndFinishesIt(t *testing.T) {
	store, clock := newTestStore(t)
	h := &recorder{clock: clock}
	pool := newTestPool(t, store, clock, h)
	ctx := context.Background()

	seedOrder(t, store, "o-1", "buyer@example.test")
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-1"})

	claimed, err := pool.ProcessOne(ctx, "w1")
	if err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}
	if !claimed {
		t.Fatal("ProcessOne reported no job, want the enqueued one")
	}
	if h.Calls() != 1 {
		t.Errorf("handler calls = %d, want 1", h.Calls())
	}
	if got := mustGet(t, store, "j1"); got.State != StateDone {
		t.Errorf("state = %q, want %q", got.State, StateDone)
	}
	if n := countRows(t, store.DB, "emails"); n != 1 {
		t.Errorf("emails = %d, want 1", n)
	}
	if n := countRows(t, store.DB, "processed_jobs"); n != 1 {
		t.Errorf("processed_jobs = %d, want 1: the effect and the record of it commit together", n)
	}
}

func TestProcessOneOnAnEmptyQueue(t *testing.T) {
	store, clock := newTestStore(t)
	h := &recorder{clock: clock}
	pool := newTestPool(t, store, clock, h)

	claimed, err := pool.ProcessOne(context.Background(), "w1")
	if err != nil {
		t.Fatalf("ProcessOne on an empty queue = %v, want no error: nothing to do is not a failure", err)
	}
	if claimed {
		t.Error("ProcessOne reported a claim on an empty queue")
	}
}

func TestFailedJobIsRetriedAfterBackoff(t *testing.T) {
	store, clock := newTestStore(t)
	h := &recorder{clock: clock, failFn: func(call int) error {
		if call == 1 {
			return errSMTP
		}
		return nil
	}}
	pool := newTestPool(t, store, clock, h)
	ctx := context.Background()

	seedOrder(t, store, "o-1", "buyer@example.test")
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-1"})

	if _, err := pool.ProcessOne(ctx, "w1"); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}

	got := mustGet(t, store, "j1")
	if got.State != StateReady {
		t.Fatalf("state = %q, want %q: a failure inside the attempt budget goes back to the queue",
			got.State, StateReady)
	}
	if got.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", got.Attempts)
	}
	if !strings.Contains(got.LastError, "smtp") {
		t.Errorf("last error = %q, want the handler's cause recorded", got.LastError)
	}
	// Backoff{Base: 1s, Rand: 0} → half of the first nominal delay.
	if want := clock.Now().Add(500 * time.Millisecond); !got.RunAt.Equal(want) {
		t.Errorf("run at = %v, want %v (now + Backoff.Delay(1)): a retry must be invisible "+
			"until its backoff has passed", got.RunAt, want)
	}
	if n := countRows(t, store.DB, "emails"); n != 0 {
		t.Errorf("emails = %d, want 0: the failed attempt's writes must roll back", n)
	}
	if n := countRows(t, store.DB, "processed_jobs"); n != 0 {
		t.Errorf("processed_jobs = %d, want 0: a marker that survives a failed attempt would "+
			"make the retry skip the work it is retrying", n)
	}

	if claimed, _ := pool.ProcessOne(ctx, "w1"); claimed {
		t.Error("ProcessOne claimed the job before its backoff elapsed")
	}

	clock.Advance(500 * time.Millisecond)
	if _, err := pool.ProcessOne(ctx, "w1"); err != nil {
		t.Fatalf("ProcessOne after the backoff: %v", err)
	}
	if got := mustGet(t, store, "j1"); got.State != StateDone || got.Attempts != 2 {
		t.Errorf("after the retry: state %q attempts %d, want %q and 2", got.State, got.Attempts, StateDone)
	}
	if n := countRows(t, store.DB, "emails"); n != 1 {
		t.Errorf("emails = %d, want 1", n)
	}
}

func TestJobDeadLettersWhenItRunsOutOfAttempts(t *testing.T) {
	store, clock := newTestStore(t)
	h := &recorder{clock: clock, failFn: func(int) error { return errSMTP }}
	pool := newTestPool(t, store, clock, h)
	ctx := context.Background()

	seedOrder(t, store, "o-1", "buyer@example.test")
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-1", MaxAttempts: 2})

	for i := 0; i < 2; i++ {
		if _, err := pool.ProcessOne(ctx, "w1"); err != nil {
			t.Fatalf("ProcessOne #%d: %v", i+1, err)
		}
		clock.Advance(time.Minute)
	}

	got := mustGet(t, store, "j1")
	if got.State != StateDead {
		t.Fatalf("state = %q after 2 of 2 attempts, want %q", got.State, StateDead)
	}
	if got.Attempts != 2 {
		t.Errorf("attempts = %d, want 2: a bounded budget is what stops a poison job from "+
			"retrying forever", got.Attempts)
	}
	if h.Calls() != 2 {
		t.Errorf("handler calls = %d, want 2", h.Calls())
	}
	if claimed, _ := pool.ProcessOne(ctx, "w1"); claimed {
		t.Error("a dead-lettered job was claimed again: nothing retries it, and nothing deletes " +
			"it either — the row is the evidence")
	}
}

func TestPermanentFailureSkipsTheAttemptBudget(t *testing.T) {
	store, clock := newTestStore(t)
	h := &recorder{clock: clock}
	pool := newTestPool(t, store, clock, h)
	ctx := context.Background()

	// No order row: the handler reports ErrPermanent, because no retry will
	// make a deleted record reappear.
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-missing"})

	if _, err := pool.ProcessOne(ctx, "w1"); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}
	got := mustGet(t, store, "j1")
	if got.State != StateDead {
		t.Errorf("state = %q, want %q: an error wrapping ErrPermanent dead-letters at once",
			got.State, StateDead)
	}
	if got.Attempts != 1 {
		t.Errorf("attempts = %d, want 1: the remaining budget is not spent on a failure "+
			"that cannot change", got.Attempts)
	}
}

func TestJobWithNoHandlerDeadLetters(t *testing.T) {
	store, clock := newTestStore(t)
	h := &recorder{clock: clock}
	pool := newTestPool(t, store, clock, h)
	ctx := context.Background()

	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "resize-image", Payload: "o-1"})

	if _, err := pool.ProcessOne(ctx, "w1"); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}
	if got := mustGet(t, store, "j1"); got.State != StateDead {
		t.Errorf("state = %q for an unregistered kind, want %q: retrying will not deploy the "+
			"missing handler", got.State, StateDead)
	}
	if h.Calls() != 0 {
		t.Errorf("handler calls = %d, want 0: the registered handler must not run someone "+
			"else's job", h.Calls())
	}
}

func TestCrashedWorkerLosesNothing(t *testing.T) {
	store, clock := newTestStore(t)
	h := &recorder{clock: clock}
	pool := newTestPool(t, store, clock, h)
	ctx := context.Background()

	seedOrder(t, store, "o-1", "buyer@example.test")
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-1"})

	// A worker claims the job and the process dies: no completion, no retry,
	// no error anywhere. All that is left is the lease.
	if _, err := store.Claim(ctx, "crashy", pool.Lease); err != nil {
		t.Fatalf("Claim: %v", err)
	}

	clock.Advance(pool.Lease + time.Second)
	claimed, err := pool.ProcessOne(ctx, "w2")
	if err != nil {
		t.Fatalf("ProcessOne after the crash: %v", err)
	}
	if !claimed {
		t.Fatal("ProcessOne found nothing after the lease expired: a crashed worker's job must " +
			"come back, or the work is silently lost")
	}
	if got := mustGet(t, store, "j1"); got.State != StateDone || got.Attempts != 2 {
		t.Errorf("state %q attempts %d, want %q and 2", got.State, got.Attempts, StateDone)
	}
	if n := countRows(t, store.DB, "emails"); n != 1 {
		t.Errorf("emails = %d, want exactly 1", n)
	}
}

func TestDuplicateDeliveryRunsTheEffectOnce(t *testing.T) {
	store, clock := newTestStore(t)
	h := &recorder{clock: clock}
	pool := newTestPool(t, store, clock, h)
	ctx := context.Background()

	seedOrder(t, store, "o-1", "buyer@example.test")
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-1"})

	// Delivery 1, by hand: the effect and its marker commit, and then the
	// worker dies before the queue row is updated. At-least-once delivery
	// means this state is not an edge case, it is the contract.
	job, err := store.Claim(ctx, "crashy", pool.Lease)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if first, err := store.MarkProcessed(ctx, tx, job.ID); err != nil || !first {
		t.Fatalf("MarkProcessed = %v, %v; want true, nil", first, err)
	}
	if err := h.Handle(ctx, tx, job); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Delivery 2: the lease expires and a healthy worker picks the job up.
	clock.Advance(pool.Lease + time.Second)
	if _, err := pool.ProcessOne(ctx, "w2"); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}

	if h.Calls() != 1 {
		t.Errorf("handler calls = %d, want 1: a redelivery of work that already committed must "+
			"not run the effect again — the customer gets one e-mail, not two", h.Calls())
	}
	if n := countRows(t, store.DB, "emails"); n != 1 {
		t.Errorf("emails = %d, want 1", n)
	}
	if got := mustGet(t, store, "j1"); got.State != StateDone {
		t.Errorf("state = %q, want %q: the duplicate delivery still has to finish the job",
			got.State, StateDone)
	}
}

func TestPoolDrainsTheQueueAcrossWorkers(t *testing.T) {
	store, clock := newTestStore(t)
	started := make(chan struct{})
	h := &recorder{clock: clock, started: started}
	pool := newTestPool(t, store, clock, h)
	pool.Workers = 4

	const jobCount = 12
	for i := 0; i < jobCount; i++ {
		id := fmt.Sprintf("o-%d", i)
		seedOrder(t, store, id, id+"@example.test")
		mustEnqueue(t, store, NewJob{ID: "j-" + id, Kind: "send-confirmation", Payload: id})
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		pool.Run(ctx)
	}()

	for i := 0; i < jobCount; i++ {
		waitFor(t, started, "a worker to reach the handler") // once per job, no more
	}
	cancel()
	waitFor(t, done, "Run to return after cancellation")

	counts, err := store.CountByState(context.Background())
	if err != nil {
		t.Fatalf("CountByState: %v", err)
	}
	if counts[StateDone] != jobCount {
		t.Errorf("done = %d of %d, want all of them (%v)", counts[StateDone], jobCount, counts)
	}
	if n := countRows(t, store.DB, "emails"); n != jobCount {
		t.Errorf("emails = %d, want %d — one per job, no more", n, jobCount)
	}
	if h.Calls() != jobCount {
		t.Errorf("handler calls = %d, want %d", h.Calls(), jobCount)
	}
}

func TestShutdownFinishesTheJobInFlight(t *testing.T) {
	store, clock := newTestStore(t)
	started := make(chan struct{})
	block := make(chan struct{})
	h := &recorder{clock: clock, started: started, block: block}
	pool := newTestPool(t, store, clock, h)

	seedOrder(t, store, "o-1", "buyer@example.test")
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-1"})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		pool.Run(ctx)
	}()

	waitFor(t, started, "the handler to start") // the job is in flight
	cancel()                                    // deploy, SIGTERM, scale-down: stop taking new work
	close(block)
	waitFor(t, done, "Run to return after cancellation")

	got := mustGet(t, store, "j1")
	if got.State != StateDone {
		t.Errorf("state = %q after shutdown, want %q: cancelling the pool must stop it "+
			"claiming, not abandon the job it already holds (context.WithoutCancel)", got.State, StateDone)
	}
	if !got.LeaseUntil.IsZero() {
		t.Errorf("lease until = %v, want it cleared: shutdown must not leave a job that "+
			"looks claimed by a process that no longer exists", got.LeaseUntil)
	}
	if n := countRows(t, store.DB, "emails"); n != 1 {
		t.Errorf("emails = %d, want 1: the in-flight effect must commit, not roll back", n)
	}
}
