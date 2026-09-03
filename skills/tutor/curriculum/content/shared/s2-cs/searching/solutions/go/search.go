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
	for i, x := range xs {
		probes++
		if x == target {
			return i, probes
		}
	}
	return -1, probes
}

// BinarySearch returns the index of some element equal to target in the
// ascending-sorted slice xs, or -1 if absent. An empty slice costs 0 probes.
func BinarySearch(xs []int, target int) (index, probes int) {
	lo, hi := 0, len(xs)
	for lo < hi {
		mid := lo + (hi-lo)/2
		probes++
		switch {
		case xs[mid] == target:
			return mid, probes
		case xs[mid] < target:
			lo = mid + 1
		default:
			hi = mid
		}
	}
	return -1, probes
}

// FirstOccurrence returns the smallest index whose element equals target in
// the ascending-sorted slice xs, or -1 if absent.
func FirstOccurrence(xs []int, target int) (index, probes int) {
	lo, probes := lowerBound(xs, target, false)
	if lo < len(xs) {
		probes++
		if xs[lo] == target {
			return lo, probes
		}
	}
	return -1, probes
}

// LastOccurrence returns the largest index whose element equals target in
// the ascending-sorted slice xs, or -1 if absent.
func LastOccurrence(xs []int, target int) (index, probes int) {
	lo, probes := lowerBound(xs, target, true)
	if lo > 0 {
		probes++
		if xs[lo-1] == target {
			return lo - 1, probes
		}
	}
	return -1, probes
}

// lowerBound returns the first index whose element is >= target, or with
// strict set, the first index whose element is > target. It may return
// len(xs) when no such index exists.
func lowerBound(xs []int, target int, strict bool) (lo, probes int) {
	hi := len(xs)
	for lo < hi {
		mid := lo + (hi-lo)/2
		probes++
		if xs[mid] < target || (strict && xs[mid] == target) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo, probes
}
