package parallel

import (
	"slices"
	"sync"
	"testing"
	"time"
)

// barrier returns an arrive func that blocks until n goroutines have all
// called it — a gate that only opens when n pieces of work run at the same
// time, so a secretly sequential implementation cannot pass. If the gate
// can never open (the work is running one piece at a time), the watchdog
// fails the run loudly instead of letting the test hang.
func barrier(t *testing.T, n int) (arrive func()) {
	t.Helper()
	var gate sync.WaitGroup
	gate.Add(n)
	watchdog := time.AfterFunc(10*time.Second, func() {
		panic("barrier never opened: the goroutines are not running concurrently — is each piece of work in its own goroutine?")
	})
	t.Cleanup(func() { watchdog.Stop() })
	return func() {
		gate.Done()
		gate.Wait()
	}
}

func TestRunAllRunsEveryTask(t *testing.T) {
	const n = 8
	done := make([]bool, n)
	tasks := make([]func(), n)
	for i := range tasks {
		tasks[i] = func() { done[i] = true }
	}
	RunAll(tasks)
	for i, ok := range done {
		if !ok {
			t.Errorf("task %d had not finished when RunAll returned — what guarantees RunAll waits for it?", i)
		}
	}
}

func TestRunAllIsConcurrent(t *testing.T) {
	const n = 8
	arrive := barrier(t, n)
	done := make([]bool, n)
	tasks := make([]func(), n)
	for i := range tasks {
		tasks[i] = func() {
			arrive() // blocks until all n tasks are running at once
			done[i] = true
		}
	}
	RunAll(tasks)
	for i, ok := range done {
		if !ok {
			t.Errorf("task %d had not finished when RunAll returned", i)
		}
	}
}

func TestRunAllEmpty(t *testing.T) {
	RunAll(nil) // must return immediately: nothing to wait for
}

func TestMapPreservesOrder(t *testing.T) {
	got := Map([]int{1, 2, 3, 4, 5}, func(x int) int { return x * x })
	want := []int{1, 4, 9, 16, 25}
	if !slices.Equal(got, want) {
		t.Errorf("Map(1..5, square) = %v, want %v — each goroutine owns out[i] and nothing else", got, want)
	}
}

func TestMapChangesType(t *testing.T) {
	got := Map([]string{"go", "routine", ""}, func(s string) int { return len(s) })
	want := []int{2, 7, 0}
	if !slices.Equal(got, want) {
		t.Errorf("Map(words, len) = %v, want %v", got, want)
	}
}

func TestMapEmpty(t *testing.T) {
	if got := Map(nil, func(x int) int { return x }); len(got) != 0 {
		t.Errorf("Map(nil, f) = %v, want an empty result", got)
	}
}

func TestMapIsConcurrent(t *testing.T) {
	const n = 8
	arrive := barrier(t, n)
	in := make([]int, n)
	for i := range in {
		in[i] = i
	}
	got := Map(in, func(x int) int {
		arrive() // blocks until all n elements are being mapped at once
		return x * 2
	})
	if len(got) != n {
		t.Fatalf("Map returned %d results, want %d", len(got), n)
	}
	for i, v := range got {
		if v != in[i]*2 {
			t.Errorf("got[%d] = %d, want %d", i, v, in[i]*2)
		}
	}
}

// seq returns 1, 2, …, n, whose sum is n*(n+1)/2.
func seq(n int) []int {
	nums := make([]int, n)
	for i := range nums {
		nums[i] = i + 1
	}
	return nums
}

func TestTotal(t *testing.T) {
	cases := []struct {
		name    string
		nums    []int
		workers int
		want    int
	}{
		{"single worker", []int{3, 1, 4}, 1, 8},
		{"even split", seq(100), 4, 5050},
		{"uneven split", seq(10), 3, 55},
		{"more workers than numbers", []int{5, 7}, 8, 12},
		{"empty input", nil, 3, 0},
		{"negative numbers", []int{-2, 9, -7}, 2, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Total(c.nums, c.workers); got != c.want {
				t.Errorf("Total(%d numbers, workers=%d) = %d, want %d — is every chunk summed exactly once?",
					len(c.nums), c.workers, got, c.want)
			}
		})
	}
}

func TestTotalUnderLoad(t *testing.T) {
	// Big enough that a shared total visibly loses updates, and that
	// go test -race reliably observes the conflicting accesses.
	const n = 100_000
	if got, want := Total(seq(n), 8), n*(n+1)/2; got != want {
		t.Errorf("Total(1..%d, workers=8) = %d, want %d — a shared counter drops increments; give each worker its own cell",
			n, got, want)
	}
}
