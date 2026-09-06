package heaps

import (
	"container/heap"
	"slices"
	"testing"
)

// checkTaskArray verifies the heap property directly on the slice:
// every parent's Priority must be <= its children's.
func checkTaskArray(t *testing.T, q TaskQueue) {
	t.Helper()
	for i := 1; i < len(q); i++ {
		parent := (i - 1) / 2
		if q[parent].Priority > q[i].Priority {
			t.Fatalf("heap property broken: q[%d].Priority=%d (parent) > q[%d].Priority=%d (child)",
				parent, q[parent].Priority, i, q[i].Priority)
		}
	}
}

func mustPopTask(t *testing.T, q *TaskQueue) Task {
	t.Helper()
	task, ok := heap.Pop(q).(Task)
	if !ok {
		t.Fatal("heap.Pop did not return a Task — your Pop method must remove and return the LAST element")
	}
	return task
}

func TestLessOrdersByAscendingPriority(t *testing.T) {
	q := TaskQueue{{Name: "deploy", Priority: 3}, {Name: "page on-call", Priority: 1}}
	if !q.Less(1, 0) {
		t.Error("Less(1, 0) = false, want true (Priority 1 is more urgent than Priority 3)")
	}
	if q.Less(0, 1) {
		t.Error("Less(0, 1) = true, want false (Priority 3 is less urgent than Priority 1)")
	}
}

func TestSwapExchangesTasks(t *testing.T) {
	q := TaskQueue{{Name: "a", Priority: 1}, {Name: "b", Priority: 2}}
	q.Swap(0, 1)
	if q[0].Name != "b" || q[1].Name != "a" {
		t.Errorf("after Swap(0, 1) queue = %v, want tasks in order b, a", q)
	}
}

func TestInitThenDrainAscending(t *testing.T) {
	q := TaskQueue{
		{Name: "compact logs", Priority: 40},
		{Name: "page on-call", Priority: 1},
		{Name: "rotate certs", Priority: 15},
		{Name: "nightly backup", Priority: 30},
		{Name: "health check", Priority: 5},
	}
	heap.Init(&q)
	checkTaskArray(t, q)
	var got []string
	for q.Len() > 0 {
		got = append(got, mustPopTask(t, &q).Name)
		checkTaskArray(t, q)
	}
	want := []string{"page on-call", "health check", "rotate certs", "nightly backup", "compact logs"}
	if !slices.Equal(got, want) {
		t.Errorf("draining the queue = %v, want ascending Priority order %v", got, want)
	}
}

func TestSchedulerScenario(t *testing.T) {
	var q TaskQueue
	heap.Push(&q, Task{Name: "reindex search", Priority: 20})
	heap.Push(&q, Task{Name: "nightly backup", Priority: 30})
	if q.Len() != 2 {
		t.Fatalf("Len() = %d after two heap.Push calls, want 2 (Push needs a pointer receiver so append survives)", q.Len())
	}
	checkTaskArray(t, q)

	if next := mustPopTask(t, &q); next.Name != "reindex search" {
		t.Fatalf("first dispatch = %q, want %q (lowest Priority runs first)", next.Name, "reindex search")
	}

	heap.Push(&q, Task{Name: "page on-call", Priority: 1})
	checkTaskArray(t, q)
	if next := mustPopTask(t, &q); next.Name != "page on-call" {
		t.Fatalf("dispatch after urgent arrival = %q, want %q (a later, more urgent task jumps the queue)", next.Name, "page on-call")
	}
	if next := mustPopTask(t, &q); next.Name != "nightly backup" {
		t.Fatalf("last dispatch = %q, want %q", next.Name, "nightly backup")
	}
	if q.Len() != 0 {
		t.Errorf("Len() = %d after draining, want 0", q.Len())
	}
}
