package dp

// FibNaive returns the n-th Fibonacci number (fib(0)=0, fib(1)=1) computed
// by plain recursion with no caching, plus the total number of calls made:
// this call and every recursive call, base cases included.
func FibNaive(n int) (value, calls int) {
	// TODO: implement per LESSON.md — translate the recurrence directly and
	// thread the call count through the two recursive results.
	return 0, 0
}

// FibMemo returns fib(n) using top-down memoization, plus the number of
// subproblems it actually computed (cache misses, base cases included).
// Each call starts with a fresh cache.
func FibMemo(n int) (value, computed int) {
	// TODO: implement — create the memo map here and recurse through a
	// helper that carries it (see the "In Go:" section of LESSON.md).
	return 0, 0
}

// FibTab returns fib(n) using bottom-up tabulation: a table filled from the
// base cases upward, no recursion.
func FibTab(n int) int {
	// TODO: implement.
	return 0
}

// FibConstSpace returns fib(n) bottom-up in O(1) extra space: no slice, no
// map, no recursion — carry only the last two values.
func FibConstSpace(n int) int {
	// TODO: implement.
	return 0
}

// MinCoins returns the fewest coins from the given denominations (each a
// positive int, usable any number of times) that sum exactly to amount, or
// -1 if no combination can. MinCoins(coins, 0) is 0.
func MinCoins(coins []int, amount int) int {
	// TODO: write the state, transition, and base cases as a comment first,
	// then tabulate bottom-up. Mind the sentinel trap from LESSON.md.
	return 0
}
