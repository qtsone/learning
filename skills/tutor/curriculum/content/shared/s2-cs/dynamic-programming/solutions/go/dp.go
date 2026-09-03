package dp

// FibNaive returns the n-th Fibonacci number (fib(0)=0, fib(1)=1) computed
// by plain recursion with no caching, plus the total number of calls made:
// this call and every recursive call, base cases included.
func FibNaive(n int) (value, calls int) {
	if n < 2 {
		return n, 1
	}
	v1, c1 := FibNaive(n - 1)
	v2, c2 := FibNaive(n - 2)
	return v1 + v2, c1 + c2 + 1
}

// FibMemo returns fib(n) using top-down memoization, plus the number of
// subproblems it actually computed (cache misses, base cases included).
// Each call starts with a fresh cache.
func FibMemo(n int) (value, computed int) {
	memo := map[int]int{}
	return fibMemo(n, memo, &computed), computed
}

func fibMemo(n int, memo map[int]int, computed *int) int {
	if v, ok := memo[n]; ok {
		return v
	}
	*computed++
	if n < 2 {
		memo[n] = n
		return n
	}
	memo[n] = fibMemo(n-1, memo, computed) + fibMemo(n-2, memo, computed)
	return memo[n]
}

// FibTab returns fib(n) using bottom-up tabulation: a table filled from the
// base cases upward, no recursion.
func FibTab(n int) int {
	if n < 2 {
		return n
	}
	table := make([]int, n+1)
	table[1] = 1
	for i := 2; i <= n; i++ {
		table[i] = table[i-1] + table[i-2]
	}
	return table[n]
}

// FibConstSpace returns fib(n) bottom-up in O(1) extra space: no slice, no
// map, no recursion — carry only the last two values.
func FibConstSpace(n int) int {
	if n < 2 {
		return n
	}
	prev2, prev1 := 0, 1
	for i := 2; i <= n; i++ {
		prev2, prev1 = prev1, prev2+prev1
	}
	return prev1
}

// MinCoins returns the fewest coins from the given denominations (each a
// positive int, usable any number of times) that sum exactly to amount, or
// -1 if no combination can. MinCoins(coins, 0) is 0.
//
// State: best[a] = fewest coins summing exactly to a.
// Transition: best[a] = 1 + min over coins c <= a of best[a-c].
// Base cases: best[0] = 0; cells no coin can reach stay at the sentinel.
func MinCoins(coins []int, amount int) int {
	// amount+1 marks "impossible": larger than any real answer, yet safe to
	// add 1 to — a max-int sentinel would wrap negative and win every min.
	impossible := amount + 1
	best := make([]int, amount+1)
	for a := 1; a <= amount; a++ {
		best[a] = impossible
		for _, c := range coins {
			if c <= a && best[a-c]+1 < best[a] {
				best[a] = best[a-c] + 1
			}
		}
	}
	if best[amount] == impossible {
		return -1
	}
	return best[amount]
}
