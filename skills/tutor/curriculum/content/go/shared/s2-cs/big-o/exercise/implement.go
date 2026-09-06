package bigo

// Part B — implement. Each function states its required complexity; the
// tests check behavior AND that requirement (via a time guard on large
// inputs, and an allocation counter for the O(1)-space one).

// HasDuplicate reports whether any value occurs more than once in xs.
// Required: O(n) time. O(n) auxiliary space is the price — pay it.
func HasDuplicate(xs []int) bool {
	// TODO: one pass, remembering every value seen so far in a map.
	return false
}

// HasDuplicateSorted reports whether any value occurs more than once in xs,
// which is guaranteed to be sorted in ascending order.
// Required: O(n) time AND O(1) auxiliary space — no maps, no new slices.
func HasDuplicateSorted(xs []int) bool {
	// TODO: sorted order puts equal values next to each other.
	return false
}

// CountCommon returns how many distinct values appear in both xs and ys.
// Duplicates inside one slice must not inflate the count.
// Required: O(len(xs) + len(ys)) time.
func CountCommon(xs, ys []int) int {
	// TODO: build a set from one slice, scan the other; count each shared
	// value only once.
	return 0
}
