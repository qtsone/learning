package dp

import (
	"slices"
	"testing"
	"time"
)

// Shared ground truth for every Fibonacci variant. fib(90) is the largest
// value the exercise uses; fib(93) would silently overflow int64.
var fibValues = []struct{ n, want int }{
	{0, 0},
	{1, 1},
	{2, 1},
	{5, 5},
	{10, 55},
	{20, 6765},
	{30, 832040},
	{45, 1134903170},
	{90, 2880067194370816120},
}

func TestFibNaiveValues(t *testing.T) {
	for _, c := range fibValues {
		if c.n > 30 {
			continue // naive recursion would take hours-to-years past here
		}
		if got, _ := FibNaive(c.n); got != c.want {
			t.Errorf("FibNaive(%d) value = %d, want %d", c.n, got, c.want)
		}
	}
}

// The call counts follow C(n) = 1 + C(n-1) + C(n-2), C(0) = C(1) = 1:
// every +5 on n multiplies the work by roughly 11. That is exponential
// growth, measured on your own code.
func TestFibNaiveCallCountExplodes(t *testing.T) {
	cases := []struct{ n, wantCalls int }{
		{0, 1},
		{1, 1},
		{10, 177},
		{15, 1973},
		{20, 21891},
		{25, 242785},
		{30, 2692537},
	}
	for _, c := range cases {
		if _, calls := FibNaive(c.n); calls != c.wantCalls {
			t.Errorf("FibNaive(%d) reported %d calls, want exactly %d (count this call plus every recursive call, base cases included)",
				c.n, calls, c.wantCalls)
		}
	}
}

// fibMemoGuard fails fast if FibMemo is secretly still exponential, so the
// n=90 test below cannot hang for years.
func fibMemoGuard(t *testing.T) {
	t.Helper()
	start := time.Now()
	_, computed := FibMemo(34)
	if computed < 1 || computed > 70 {
		t.Fatalf("FibMemo(34) computed %d subproblems, want between 1 and 70: check the cache before computing and store after — fix that before the n=90 case runs", computed)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("FibMemo(34) took %v — that is exponential territory; the cache is not being reused", elapsed)
	}
}

func TestFibMemoValues(t *testing.T) {
	fibMemoGuard(t)
	for _, c := range fibValues {
		if got, _ := FibMemo(c.n); got != c.want {
			t.Errorf("FibMemo(%d) value = %d, want %d", c.n, got, c.want)
		}
	}
}

func TestFibMemoComputesEachSubproblemOnce(t *testing.T) {
	fibMemoGuard(t)
	cases := []struct{ n, maxComputed int }{
		{10, 22},
		{30, 62},
		{60, 122},
	}
	for _, c := range cases {
		if _, computed := FibMemo(c.n); computed < 1 || computed > c.maxComputed {
			t.Errorf("FibMemo(%d) computed %d subproblems, want between 1 and %d: there are only %d distinct subproblems, and memoization computes each at most once (naive recursion at n=30 alone makes 2,692,537 calls)",
				c.n, computed, c.maxComputed, c.n+1)
		}
	}
}

func TestFibTabValues(t *testing.T) {
	for _, c := range fibValues {
		if got := FibTab(c.n); got != c.want {
			t.Errorf("FibTab(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

func TestFibConstSpaceValues(t *testing.T) {
	for _, c := range fibValues {
		if got := FibConstSpace(c.n); got != c.want {
			t.Errorf("FibConstSpace(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

func TestMinCoins(t *testing.T) {
	cases := []struct {
		name   string
		coins  []int
		amount int
		want   int
	}{
		{"amount zero needs no coins", []int{1, 5}, 0, 0},
		{"single coin exact", []int{5}, 5, 1},
		{"greedy trap: 3+3 beats 4+1+1", []int{1, 3, 4}, 6, 2},
		{"us coins", []int{1, 5, 10, 25}, 63, 6},
		{"odd amount with only twos", []int{2}, 7, -1},
		{"amount below every coin", []int{5, 7}, 3, -1},
		{"two of each", []int{5, 7}, 24, 4},
		{"coin larger than amount ignored", []int{3, 50}, 9, 3},
		{"larger amount", []int{1, 2, 5}, 100, 20},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			coins := slices.Clone(c.coins)
			if got := MinCoins(coins, c.amount); got != c.want {
				t.Errorf("MinCoins(%v, %d) = %d, want %d", c.coins, c.amount, got, c.want)
			}
		})
	}
}

// O(amount × #coins) handles this instantly; anything exponential in the
// amount would never finish. Note it says nothing about the "impossible"
// sentinel: the 1-coin makes every cell reachable, so the sentinel is never
// read here — an overflowing math.MaxInt sentinel is what breaks the small
// "{2}, amount 7" case above.
func TestMinCoinsLargeAmount(t *testing.T) {
	if got, want := MinCoins([]int{1, 5, 12, 19}, 10000), 528; got != want {
		t.Errorf("MinCoins([1 5 12 19], 10000) = %d, want %d", got, want)
	}
}
