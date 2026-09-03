package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestEnqueueAppliesDefaults(t *testing.T) {
	store, clock := newTestStore(t)
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-1"})

	job := mustGet(t, store, "j1")
	if job.State != StateReady {
		t.Errorf("state = %q, want %q", job.State, StateReady)
	}
	if job.Attempts != 0 {
		t.Errorf("attempts = %d, want 0 before the job is claimed", job.Attempts)
	}
	if job.MaxAttempts != DefaultMaxAttempts {
		t.Errorf("max attempts = %d, want DefaultMaxAttempts (%d) when the producer names none",
			job.MaxAttempts, DefaultMaxAttempts)
	}
	if !job.RunAt.Equal(clock.Now()) {
		t.Errorf("run at = %v, want the clock's now (%v): a job with no RunAt is due immediately",
			job.RunAt, clock.Now())
	}
	if !job.LeaseUntil.IsZero() {
		t.Errorf("lease until = %v, want the zero time on an unclaimed job", job.LeaseUntil)
	}
}

func TestEnqueueRejectsIncompleteJob(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	for _, j := range []NewJob{
		{ID: "", Kind: "send-confirmation"},
		{ID: "j1", Kind: ""},
	} {
		tx, err := store.DB.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		if err := store.Enqueue(ctx, tx, j); err == nil {
			t.Errorf("Enqueue(%+v) = nil, want an error: a job needs an id and a kind", j)
		}
		tx.Rollback()
	}
}

func TestClaimEmptyQueue(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.Claim(context.Background(), "w1", time.Minute)
	if !errors.Is(err, ErrNoJob) {
		t.Fatalf("Claim on an empty queue = %v, want ErrNoJob", err)
	}
}

func TestClaimLeasesTheJob(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-1"})

	job, err := store.Claim(ctx, "w1", 30*time.Second)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if job.ID != "j1" || job.Payload != "o-1" {
		t.Errorf("claimed %+v, want the job that was enqueued", job)
	}
	if job.Attempts != 1 {
		t.Errorf("attempts = %d, want 1: the attempt is counted when the job is claimed, "+
			"so a worker that dies mid-job still spends one", job.Attempts)
	}

	stored := mustGet(t, store, "j1")
	if stored.State != StateRunning {
		t.Errorf("stored state = %q, want %q", stored.State, StateRunning)
	}
	if want := clock.Now().Add(30 * time.Second); !stored.LeaseUntil.Equal(want) {
		t.Errorf("lease until = %v, want %v (now + lease)", stored.LeaseUntil, want)
	}
	if stored.Worker != "w1" {
		t.Errorf("worker = %q, want %q", stored.Worker, "w1")
	}

	if _, err := store.Claim(ctx, "w2", 30*time.Second); !errors.Is(err, ErrNoJob) {
		t.Errorf("second Claim = %v, want ErrNoJob: a leased job is invisible to other workers", err)
	}
}

func TestClaimSkipsJobsScheduledForLater(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	mustEnqueue(t, store, NewJob{
		ID: "j1", Kind: "send-confirmation", Payload: "o-1",
		RunAt: clock.Now().Add(time.Minute),
	})

	if _, err := store.Claim(ctx, "w1", time.Minute); !errors.Is(err, ErrNoJob) {
		t.Fatalf("Claim before RunAt = %v, want ErrNoJob: a delayed job is not due yet", err)
	}

	clock.Advance(time.Minute)
	if _, err := store.Claim(ctx, "w1", time.Minute); err != nil {
		t.Fatalf("Claim at RunAt: %v, want the job to become claimable", err)
	}
}

func TestClaimRedeliversAfterLeaseExpiry(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	mustEnqueue(t, store, NewJob{ID: "j1", Kind: "send-confirmation", Payload: "o-1"})

	if _, err := store.Claim(ctx, "crashy", 30*time.Second); err != nil {
		t.Fatalf("first Claim: %v", err)
	}
	if _, err := store.Claim(ctx, "w2", 30*time.Second); !errors.Is(err, ErrNoJob) {
		t.Fatalf("Claim while the lease holds = %v, want ErrNoJob", err)
	}

	clock.Advance(31 * time.Second)
	job, err := store.Claim(ctx, "w2", 30*time.Second)
	if err != nil {
		t.Fatalf("Claim after the lease expired: %v, want redelivery — the worker that "+
			"held it is gone and nothing else will notice", err)
	}
	if job.Attempts != 2 {
		t.Errorf("attempts = %d, want 2: the crashed delivery counts", job.Attempts)
	}
	if job.Worker != "w2" {
		t.Errorf("worker = %q, want %q: the claim moves to the new owner", job.Worker, "w2")
	}
}

func TestClaimTakesTheOldestDueJobFirst(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	mustEnqueue(t, store, NewJob{ID: "late", Kind: "send-confirmation", RunAt: clock.Now().Add(time.Second)})
	mustEnqueue(t, store, NewJob{ID: "early", Kind: "send-confirmation", RunAt: clock.Now()})

	clock.Advance(time.Second)
	job, err := store.Claim(ctx, "w1", time.Minute)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if job.ID != "early" {
		t.Errorf("claimed %q, want %q: due jobs come out oldest run_at first", job.ID, "early")
	}
}

func TestCompleteRetryAndDeadLetter(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()

	mustEnqueue(t, store, NewJob{ID: "done", Kind: "send-confirmation"})
	mustEnqueue(t, store, NewJob{ID: "retried", Kind: "send-confirmation"})
	mustEnqueue(t, store, NewJob{ID: "dead", Kind: "send-confirmation"})

	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := store.Complete(ctx, tx, "done"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if got := mustGet(t, store, "done"); got.State != StateDone || !got.LeaseUntil.IsZero() {
		t.Errorf("after Complete: state %q lease %v, want %q and no lease",
			got.State, got.LeaseUntil, StateDone)
	}

	runAt := clock.Now().Add(2 * time.Second)
	if err := store.Retry(ctx, "retried", runAt, "smtp timeout"); err != nil {
		t.Fatalf("Retry: %v", err)
	}
	got := mustGet(t, store, "retried")
	if got.State != StateReady || !got.RunAt.Equal(runAt) || got.LastError != "smtp timeout" {
		t.Errorf("after Retry: %+v, want state %q, run at %v, last error %q",
			got, StateReady, runAt, "smtp timeout")
	}
	if !got.LeaseUntil.IsZero() {
		t.Errorf("after Retry: lease until %v, want it cleared so any worker can take the job",
			got.LeaseUntil)
	}

	if err := store.DeadLetter(ctx, "dead", "invalid address"); err != nil {
		t.Fatalf("DeadLetter: %v", err)
	}
	if got := mustGet(t, store, "dead"); got.State != StateDead || got.LastError != "invalid address" {
		t.Errorf("after DeadLetter: state %q last error %q, want %q and the cause recorded",
			got.State, got.LastError, StateDead)
	}

	clock.Advance(time.Hour)
	claimed := map[string]bool{}
	for {
		job, err := store.Claim(ctx, "w1", time.Minute)
		if errors.Is(err, ErrNoJob) {
			break
		}
		if err != nil {
			t.Fatalf("Claim: %v", err)
		}
		claimed[job.ID] = true
	}
	if len(claimed) != 1 || !claimed["retried"] {
		t.Errorf("claimable jobs = %v, want only the retried one: done and dead jobs never come back", claimed)
	}
}

func TestUpdatingAnUnknownJobIsAnError(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()

	if err := store.Retry(ctx, "ghost", clock.Now(), "x"); err == nil {
		t.Error("Retry on an unknown job = nil, want an error rather than a silent no-op")
	}
	if err := store.DeadLetter(ctx, "ghost", "x"); err == nil {
		t.Error("DeadLetter on an unknown job = nil, want an error rather than a silent no-op")
	}
}

func TestMarkProcessedIsTrueOnlyOnce(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	for i, want := range []bool{true, false, false} {
		tx, err := store.DB.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		got, err := store.MarkProcessed(ctx, tx, "j1")
		if err != nil {
			t.Fatalf("MarkProcessed #%d: %v", i+1, err)
		}
		if got != want {
			t.Errorf("MarkProcessed #%d = %v, want %v: only the first delivery may run the effect",
				i+1, got, want)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}
	if n := countRows(t, store.DB, "processed_jobs"); n != 1 {
		t.Errorf("processed_jobs rows = %d, want 1", n)
	}
}

func TestMarkProcessedRollsBackWithItsTransaction(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := store.MarkProcessed(ctx, tx, "j1"); err != nil {
		t.Fatalf("MarkProcessed: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	tx2, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx2.Rollback()
	got, err := store.MarkProcessed(ctx, tx2, "j1")
	if err != nil {
		t.Fatalf("MarkProcessed: %v", err)
	}
	if !got {
		t.Error("MarkProcessed after a rollback = false, want true: a rolled-back effect " +
			"must not leave a marker that makes the retry skip the work")
	}
}

func TestConcurrentClaimsNeverHandOutTheSameJobTwice(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	const jobCount = 12
	for i := 0; i < jobCount; i++ {
		mustEnqueue(t, store, NewJob{ID: string(rune('a' + i)), Kind: "send-confirmation"})
	}

	var (
		mu     sync.Mutex
		claims = map[string]int{}
		wg     sync.WaitGroup
	)
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for {
				job, err := store.Claim(ctx, "w", time.Minute)
				if errors.Is(err, ErrNoJob) {
					return
				}
				if err != nil {
					t.Errorf("Claim: %v", err)
					return
				}
				mu.Lock()
				claims[job.ID]++
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	if len(claims) != jobCount {
		t.Errorf("claimed %d distinct jobs, want %d", len(claims), jobCount)
	}
	for id, n := range claims {
		if n != 1 {
			t.Errorf("job %q claimed %d times, want 1: finding and leasing a job must be "+
				"one statement, or two workers pick the same row", id, n)
		}
	}
}
