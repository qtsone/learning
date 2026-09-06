// Package search implements linear and binary search from scratch.
//
// Every function returns the index it found (or -1) plus probes: the number
// of elements of xs it examined. Count one probe each time you inspect an
// element at some index. The tests check both the answer and the work done.
package search

// LinearSearch scans xs from the front and returns the index of the first
// element equal to target, or -1 if absent. xs may be in any order.
// A hit at index i costs exactly i+1 probes; a miss costs len(xs) probes.
func LinearSearch(xs []int, target int) (index, probes int) {
	// TODO: scan left to right and stop at the first match.
	return -1, 0
}

// BinarySearch returns the index of some element equal to target in the
// ascending-sorted slice xs, or -1 if absent. An empty slice costs 0 probes.
func BinarySearch(xs []int, target int) (index, probes int) {
	// TODO: keep a half-open interval [lo, hi) and halve it every iteration.
	return -1, 0
}

// FirstOccurrence returns the smallest index whose element equals target in
// the ascending-sorted slice xs, or -1 if absent.
func FirstOccurrence(xs []int, target int) (index, probes int) {
	// TODO: find the first index with xs[i] >= target (keep halving — do not
	// walk sideways from a match), then verify it holds the target.
	return -1, 0
}

// LastOccurrence returns the largest index whose element equals target in
// the ascending-sorted slice xs, or -1 if absent.
func LastOccurrence(xs []int, target int) (index, probes int) {
	// TODO: find the first index with xs[i] > target; the answer, if any,
	// sits one to its left.
	return -1, 0
}
