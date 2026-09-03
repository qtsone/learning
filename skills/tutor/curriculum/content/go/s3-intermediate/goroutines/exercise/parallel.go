// Package parallel is a tiny fan-out toolkit: run independent pieces of
// work concurrently and return only when every result is in.
package parallel

// RunAll runs every task in its own goroutine and returns only after all
// of them have finished.
func RunAll(tasks []func()) {
	// TODO: one goroutine per task, a sync.WaitGroup for completion.
	// Add before go; defer Done inside the goroutine.
}

// Map applies f to every element of in — one goroutine per element — and
// returns the results in input order: out[i] == f(in[i]).
func Map[T, R any](in []T, f func(T) R) []R {
	// TODO: preallocate the result slice, let each goroutine own exactly
	// one index, Wait, then return. Distinct slice elements are distinct
	// memory — that is what keeps the writers race-free.
	return nil
}

// Total returns the sum of nums, computed by workers goroutines that each
// sum one contiguous chunk. workers is at least 1 and may exceed
// len(nums), in which case the extra workers get empty chunks.
func Total(nums []int, workers int) int {
	// TODO: partition nums, give each worker its own subtotal cell,
	// combine the cells after Wait. Per the lesson: write the shared-total
	// version first, run go test -race, read the report — then fix it.
	return 0
}
