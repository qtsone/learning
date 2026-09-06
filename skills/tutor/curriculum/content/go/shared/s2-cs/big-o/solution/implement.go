package bigo

// Part B — implement. Each function states its required complexity; the
// tests check behavior AND that requirement (via a time guard on large
// inputs, and an allocation counter for the O(1)-space one).

// HasDuplicate reports whether any value occurs more than once in xs.
// Required: O(n) time. O(n) auxiliary space is the price — pay it.
func HasDuplicate(xs []int) bool {
	seen := make(map[int]bool, len(xs))
	for _, x := range xs {
		if seen[x] {
			return true
		}
		seen[x] = true
	}
	return false
}

// HasDuplicateSorted reports whether any value occurs more than once in xs,
// which is guaranteed to be sorted in ascending order.
// Required: O(n) time AND O(1) auxiliary space — no maps, no new slices.
func HasDuplicateSorted(xs []int) bool {
	for i := 1; i < len(xs); i++ {
		if xs[i] == xs[i-1] {
			return true
		}
	}
	return false
}

// CountCommon returns how many distinct values appear in both xs and ys.
// Duplicates inside one slice must not inflate the count.
// Required: O(len(xs) + len(ys)) time.
func CountCommon(xs, ys []int) int {
	inXs := make(map[int]bool, len(xs))
	for _, x := range xs {
		inXs[x] = true
	}
	count := 0
	for _, y := range ys {
		if inXs[y] {
			count++
			// Each shared value counts once, however often it repeats in ys.
			delete(inXs, y)
		}
	}
	return count
}
