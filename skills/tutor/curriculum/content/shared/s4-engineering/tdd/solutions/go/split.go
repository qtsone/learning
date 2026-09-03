package tdd

// SplitEvenly splits totalCents (>= 0) into n (>= 1) shares that sum to
// totalCents, with no two shares differing by more than one cent and the
// larger shares first.
func SplitEvenly(totalCents, n int) []int {
	shares := make([]int, n)
	base := totalCents / n
	extra := totalCents % n
	for i := range shares {
		shares[i] = base
		if i < extra {
			shares[i]++
		}
	}
	return shares
}
