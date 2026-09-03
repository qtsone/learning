package closures

// Counter returns an increment function and a reset function that share one
// captured count. inc increments the count and returns the new value; reset
// sets it back to zero. Each call to Counter creates fresh, independent state.
func Counter() (inc func() int, reset func()) {
	// TODO: declare the count here and return two closures that share it.
	return func() int { return 0 }, func() {}
}
