package closures

// MakeAdders returns one function per offset: the function at index i returns
// its argument plus offsets[i].
func MakeAdders(offsets []int) []func(int) int {
	adders := make([]func(int) int, 0, len(offsets))
	// Safe since Go 1.22: each iteration gets a fresh off, so every closure
	// captures its own variable.
	for _, off := range offsets {
		adders = append(adders, func(n int) int { return n + off })
	}
	return adders
}

// Filter returns a new slice holding the elements of s for which keep returns
// true, preserving their original order.
func Filter[T any](s []T, keep func(T) bool) []T {
	var out []T
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}
