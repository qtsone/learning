package main

// Sign reports whether n is "negative", "zero", or "positive".
func Sign(n int) string {
	// TODO: an if / else if / else chain (criterion 1).
	return ""
}

// Award returns the medal for a podium place:
// 1 → "gold", 2 → "silver", 3 → "bronze", anything else → "none".
func Award(place int) string {
	// TODO: a switch with a default case (criterion 2).
	return ""
}

// SumEvens returns the sum of the even numbers from 1 through limit.
func SumEvens(limit int) int {
	// TODO: a classic three-part for loop; skip odd numbers with continue
	// (criteria 3 and 8).
	return 0
}

// Repeat returns word concatenated times times; "" when times <= 0.
func Repeat(word string, times int) string {
	// TODO: a range-over-integer loop (criterion 4).
	return ""
}

// CollatzSteps returns how many steps n (>= 1) takes to reach 1, where a
// step halves an even number and turns an odd n into 3n+1.
func CollatzSteps(n int) int {
	// TODO: a condition-only for loop (criterion 5).
	return 0
}

// FirstPowerAbove returns the smallest power of two strictly greater
// than limit.
func FirstPowerAbove(limit int) int {
	// TODO: an infinite for loop that exits with break (criterion 6).
	return 0
}

// CountPrimes returns how many primes exist from 2 through limit.
// A prime is divisible only by 1 and itself.
func CountPrimes(limit int) int {
	// TODO: nested loops; reject a non-prime candidate with a labeled
	// continue (criterion 7).
	return 0
}
