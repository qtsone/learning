package jobs

import (
	"context"
	"testing"
)

func orderCount(t *testing.T, store *Store, id string) int {
	t.Helper()
	var n int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	return n
}

func TestCreateOrderCommitsTheOrderAndTheJobTogether(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()

	order := Order{ID: "o-1", Email: "buyer@example.test", TotalCents: 2500}
	job := NewJob{ID: "send-confirmation:o-1", Kind: "send-confirmation", Payload: "o-1"}
	if err := CreateOrder(ctx, store, order, job); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	if n := orderCount(t, store, "o-1"); n != 1 {
		t.Errorf("orders = %d, want 1", n)
	}
	got := mustGet(t, store, "send-confirmation:o-1")
	if got.Kind != "send-confirmation" || got.Payload != "o-1" || got.State != StateReady {
		t.Errorf("job = %+v, want a ready send-confirmation job for o-1", got)
	}

	// The job is immediately claimable, and the row it refers to is already
	// visible to the worker — both landed in the same commit.
	h := &recorder{clock: clock}
	pool := newTestPool(t, store, clock, h)
	if _, err := pool.ProcessOne(ctx, "w1"); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}
	if h.Calls() != 1 {
		t.Errorf("handler calls = %d, want 1", h.Calls())
	}
}

func TestCreateOrderWritesNothingWhenTheEnqueueFails(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	// The job id is already taken, so the second statement of the transaction
	// fails. Neither half may survive.
	mustEnqueue(t, store, NewJob{ID: "send-confirmation:o-1", Kind: "send-confirmation", Payload: "o-1"})

	order := Order{ID: "o-1", Email: "buyer@example.test", TotalCents: 2500}
	job := NewJob{ID: "send-confirmation:o-1", Kind: "send-confirmation", Payload: "o-1"}
	if err := CreateOrder(ctx, store, order, job); err == nil {
		t.Fatal("CreateOrder = nil, want the duplicate job id to fail the whole operation")
	}

	if n := orderCount(t, store, "o-1"); n != 0 {
		t.Errorf("orders = %d, want 0: an order that could not enqueue its work must not exist. "+
			"Insert-then-publish would have left this row behind with no job", n)
	}
	if n := countRows(t, store.DB, "jobs"); n != 1 {
		t.Errorf("jobs = %d, want only the pre-existing one", n)
	}
}

func TestEnqueueRollsBackWithTheBusinessWrite(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	// The mirror image: the business transaction is abandoned after the job
	// was enqueued inside it. Publishing to a broker first would have left a
	// phantom job whose order never existed.
	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO orders (id, email, total_cents) VALUES (?, ?, ?)`,
		"o-2", "buyer@example.test", 500); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	if err := store.Enqueue(ctx, tx, NewJob{ID: "j2", Kind: "send-confirmation", Payload: "o-2"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if n := orderCount(t, store, "o-2"); n != 0 {
		t.Errorf("orders = %d, want 0", n)
	}
	if n := countRows(t, store.DB, "jobs"); n != 0 {
		t.Errorf("jobs = %d, want 0: a job enqueued inside a transaction that rolled back "+
			"must roll back with it", n)
	}
}
