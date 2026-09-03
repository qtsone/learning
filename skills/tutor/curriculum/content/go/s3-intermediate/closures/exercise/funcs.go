package closures

// MakeAdders returns one function per offset: the function at index i returns
// its argument plus offsets[i].
func MakeAdders(offsets []int) []func(int) int {
	// TODO: build a slice with one closure per offset. Each closure must
	// capture its own offset — recall the lesson's loop-variable section.
	return nil
}

// Filter returns a new slice holding the elements of s for which keep returns
// true, preserving their original order.
func Filter[T any](s []T, keep func(T) bool) []T {
	// TODO: loop over s and collect the elements keep approves.
	return nil
}
