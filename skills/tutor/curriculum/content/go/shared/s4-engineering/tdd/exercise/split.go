package tdd

// SplitEvenly splits totalCents (>= 0) into n (>= 1) shares that sum to
// totalCents, with no two shares differing by more than one cent and the
// larger shares first.
//
// Part C: this implementation has a bug. Write a failing test in a new
// split_test.go that exposes it, watch the test fail, and only then fix
// the code here.
func SplitEvenly(totalCents, n int) []int {
	shares := make([]int, n)
	for i := range shares {
		shares[i] = totalCents / n
	}
	return shares
}
