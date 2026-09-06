package sq

import (
	"testing"
	"time"
)

func TestQueueFIFO(t *testing.T) {
	var q Queue
	for i := 1; i <= 3; i++ {
		q.Enqueue(i)
	}
	for want := 1; want <= 3; want++ {
		got, ok := q.Dequeue()
		if !ok {
			t.Fatalf("Dequeue: queue reported empty, want %d", want)
		}
		if got != want {
			t.Errorf("Dequeue = %d, want %d (first in must come out first)", got, want)
		}
	}
}

func TestQueueEmpty(t *testing.T) {
	var q Queue
	if v, ok := q.Dequeue(); ok {
		t.Errorf("Dequeue on empty queue = (%d, true), want ok = false", v)
	}
	if v, ok := q.Peek(); ok {
		t.Errorf("Peek on empty queue = (%d, true), want ok = false", v)
	}
}

func TestQueuePeekDoesNotRemove(t *testing.T) {
	var q Queue
	q.Enqueue(7)
	q.Enqueue(8)
	for i := 1; i <= 2; i++ {
		got, ok := q.Peek()
		if !ok || got != 7 {
			t.Fatalf("Peek call #%d = (%d, %v), want (7, true)", i, got, ok)
		}
	}
	if q.Len() != 2 {
		t.Errorf("Len after two Peeks = %d, want 2 (Peek must not remove)", q.Len())
	}
}

func TestQueueLen(t *testing.T) {
	var q Queue
	if q.Len() != 0 {
		t.Errorf("Len of new queue = %d, want 0", q.Len())
	}
	q.Enqueue(1)
	q.Enqueue(2)
	if q.Len() != 2 {
		t.Errorf("Len after two enqueues = %d, want 2", q.Len())
	}
	q.Dequeue()
	if q.Len() != 1 {
		t.Errorf("Len after one dequeue = %d, want 1", q.Len())
	}
}

// Draining to empty and enqueuing again catches the classic bug of not
// resetting tail when the last element leaves.
func TestQueueDrainThenReuse(t *testing.T) {
	var q Queue
	q.Enqueue(1)
	if _, ok := q.Dequeue(); !ok {
		t.Fatal("Dequeue on a one-element queue reported empty")
	}
	q.Enqueue(2)
	got, ok := q.Dequeue()
	if !ok {
		t.Fatal("Dequeue after drain-and-reuse reported empty (was tail reset when the queue drained?)")
	}
	if got != 2 {
		t.Errorf("Dequeue after drain-and-reuse = %d, want 2", got)
	}
}

// A linked-node queue finishes this in a few milliseconds; an implementation
// that shifts every remaining element on each Dequeue does billions of moves.
// The guard is hundreds of times the honest O(1)-per-operation cost.
func TestQueueLargeWorkload(t *testing.T) {
	const n = 100_000
	const timeGuard = 500 * time.Millisecond
	var q Queue
	start := time.Now()
	for i := 0; i < n; i++ {
		q.Enqueue(i)
	}
	if q.Len() != n {
		t.Fatalf("Len after %d enqueues = %d, want %d", n, q.Len(), n)
	}
	for i := 0; i < n; i++ {
		got, ok := q.Dequeue()
		if !ok || got != i {
			t.Fatalf("Dequeue #%d = (%d, %v), want (%d, true)", i, got, ok, i)
		}
	}
	elapsed := time.Since(start)
	if _, ok := q.Dequeue(); ok {
		t.Error("queue should be empty after draining")
	}
	if elapsed > timeGuard {
		t.Errorf("%d enqueues + %d dequeues took %v — shifting the remaining elements on every Dequeue is O(n^2); link nodes (or advance a front index) so each operation is O(1)", n, n, elapsed)
	}
}
