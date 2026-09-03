package patterns

// Result pairs a job with the value fn produced for it. Results are
// collected in completion order — the Job field is how callers recover
// which job a value belongs to.
type Result struct {
	Job int
	Val int
}

// RunPool processes every job with at most `workers` concurrent calls to fn
// and returns one Result per job. workers is at least 1. It must not leak
// goroutines: by the time it returns, the feeder, the workers, and the
// closer have all exited.
//
// TODO: build the four-role choreography from the lesson —
//   - a jobs channel fed (and closed) by a feeder goroutine,
//   - exactly `workers` worker goroutines ranging over it,
//   - a closer goroutine that waits for the workers, then closes results,
//   - the calling goroutine collecting from results until it closes.
func RunPool(workers int, jobs []int, fn func(int) int) []Result {
	return nil
}
