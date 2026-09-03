package patterns

import "sync"

// Result pairs a job with the value fn produced for it. Results are
// collected in completion order — the Job field is how callers recover
// which job a value belongs to.
type Result struct {
	Job int
	Val int
}

// RunPool processes every job with at most `workers` concurrent calls to fn
// and returns one Result per job. workers is at least 1. It does not leak
// goroutines: by the time it returns, the feeder, the workers, and the
// closer have all exited.
func RunPool(workers int, jobs []int, fn func(int) int) []Result {
	jobsCh := make(chan int)
	results := make(chan Result)

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for j := range jobsCh {
				results <- Result{Job: j, Val: fn(j)}
			}
		}()
	}

	// Feeder runs in its own goroutine so the collector below can start
	// receiving immediately; feeding and collecting from one goroutine
	// deadlocks once the workers fill up.
	go func() {
		for _, j := range jobs {
			jobsCh <- j
		}
		close(jobsCh)
	}()

	// Closer: results can close only after every worker stops sending.
	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]Result, 0, len(jobs))
	for r := range results {
		out = append(out, r)
	}
	return out
}
