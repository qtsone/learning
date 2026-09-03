package closures

// Counter returns an increment function and a reset function that share one
// captured count. inc increments the count and returns the new value; reset
// sets it back to zero. Each call to Counter creates fresh, independent state.
func Counter() (inc func() int, reset func()) {
	count := 0
	inc = func() int {
		count++
		return count
	}
	reset = func() {
		count = 0
	}
	return inc, reset
}
