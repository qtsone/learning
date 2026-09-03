// Package parallel is a tiny fan-out toolkit: run independent pieces of
// work concurrently and return only when every result is in.
package parallel

import "sync"

// RunAll runs every task in its own goroutine and returns only after all
// of them have finished.
func RunAll(tasks []func()) {
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			task()
		}()
	}
	wg.Wait()
}

// Map applies f to every element of in — one goroutine per element — and
// returns the results in input order: out[i] == f(in[i]).
func Map[T, R any](in []T, f func(T) R) []R {
	out := make([]R, len(in))
	var wg sync.WaitGroup
	// Add(len) up front is as correct as Add(1) per iteration — both
	// happen before the go statements and hence before Wait.
	wg.Add(len(in))
	for i, v := range in {
		go func() {
			defer wg.Done()
			// This goroutine owns out[i] exclusively; the caller reads it
			// only after Wait, so there is no unsynchronized sharing.
			out[i] = f(v)
		}()
	}
	wg.Wait()
	return out
}

// Total returns the sum of nums, computed by workers goroutines that each
// sum one contiguous chunk. workers is at least 1 and may exceed
// len(nums), in which case the extra workers get empty chunks.
func Total(nums []int, workers int) int {
	// One cell per worker instead of one shared total: cells are written
	// by exactly one goroutine each and combined only after Wait.
	subtotals := make([]int, workers)
	chunk := (len(nums) + workers - 1) / workers
	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := min(w*chunk, len(nums))
			end := min(start+chunk, len(nums))
			for _, n := range nums[start:end] {
				subtotals[w] += n
			}
		}()
	}
	wg.Wait()
	total := 0
	for _, s := range subtotals {
		total += s
	}
	return total
}
