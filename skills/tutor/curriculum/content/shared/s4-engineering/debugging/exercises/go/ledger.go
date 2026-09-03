// Package ledger computes summary statistics for one day of expense
// amounts (purchases positive, refunds negative).
//
// Debugging exercise: all three functions are fully implemented, and each
// contains exactly one seeded bug. Do not rewrite them — reproduce,
// minimize, hypothesize, and make the smallest fix (see LESSON.md), keeping
// your trail in debugging-log.md.
package ledger

import "slices"

// batchSize mirrors how amounts arrive from the importer: in blocks of four.
const batchSize = 4

// Lowest returns the smallest amount in xs.
// It panics if xs is empty (callers guarantee at least one amount).
func Lowest(xs []float64) float64 {
	lowest := 0.0
	for _, x := range xs {
		if x < lowest {
			lowest = x
		}
	}
	return lowest
}

// Total returns the sum of all amounts in xs.
func Total(xs []float64) float64 {
	var total float64
	for i := 0; i+batchSize <= len(xs); i += batchSize {
		for _, x := range xs[i : i+batchSize] {
			total += x
		}
	}
	return total
}

// Median returns the middle amount: the middle value for an odd count, the
// mean of the two middle values for an even count.
// It panics if xs is empty (callers guarantee at least one amount).
func Median(xs []float64) float64 {
	slices.Sort(xs)
	n := len(xs)
	if n%2 == 1 {
		return xs[n/2]
	}
	return (xs[n/2-1] + xs[n/2]) / 2
}
