// Package ledger computes summary statistics for one day of expense
// amounts (purchases positive, refunds negative).
package ledger

import "slices"

// Lowest returns the smallest amount in xs.
// It panics if xs is empty (callers guarantee at least one amount).
func Lowest(xs []float64) float64 {
	// Seeded bug was `lowest := 0.0`: a phantom amount that no all-positive
	// day could ever beat. Start from real data, not a made-up sentinel.
	lowest := xs[0]
	for _, x := range xs[1:] {
		if x < lowest {
			lowest = x
		}
	}
	return lowest
}

// Total returns the sum of all amounts in xs.
//
// The starter walked xs in fixed-size batches and its loop condition dropped
// the final partial batch. Summing needs no batches at all, so the simplest
// correct loop replaces the whole walk. (Clamping the batch end with
// min(i+batchSize, len(xs)) is an equally correct minimal fix.)
func Total(xs []float64) float64 {
	var total float64
	for _, x := range xs {
		total += x
	}
	return total
}

// Median returns the middle amount: the middle value for an odd count, the
// mean of the two middle values for an even count.
// It panics if xs is empty (callers guarantee at least one amount).
func Median(xs []float64) float64 {
	// Seeded bug sorted xs in place, silently reordering the caller's slice
	// (action at a distance). Sort a clone; leave the input alone.
	sorted := slices.Clone(xs)
	slices.Sort(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}
