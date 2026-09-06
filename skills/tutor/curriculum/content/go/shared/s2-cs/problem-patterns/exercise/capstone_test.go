package patterns

import (
	"testing"
	"time"
)

func TestBalancedBrackets(t *testing.T) {
	cases := []struct {
		name string
		s    string
		want bool
	}{
		{"empty", "", true},
		{"single pair", "()", true},
		{"all three kinds nested", "([{}])", true},
		{"sequence of pairs", "()[]{}", true},
		{"wrong closer", "(]", false},
		{"interleaved", "([)]", false},
		{"unclosed opener", "(((", false},
		{"stray closer", "())", false},
		{"closer first", "]", false},
		{"non-brackets ignored", "fn(a[i]) { return b[j] }", true},
		{"non-brackets ignored, still broken", "if (x { y )", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BalancedBrackets(c.s); got != c.want {
				t.Errorf("BalancedBrackets(%q) = %v, want %v", c.s, got, c.want)
			}
		})
	}
}

func grid(rows ...string) [][]byte {
	g := make([][]byte, len(rows))
	for i, r := range rows {
		g[i] = []byte(r)
	}
	return g
}

func TestCountIslands(t *testing.T) {
	cases := []struct {
		name string
		rows []string
		want int
	}{
		{"no grid", nil, 0},
		{"all water", []string{"...", "..."}, 0},
		{"all land", []string{"##", "##"}, 1},
		{"single cell island", []string{"#"}, 1},
		{"diagonals do not connect", []string{"#.", ".#"}, 2},
		{"three islands", []string{"#..#", "#...", "..##"}, 3},
		{"snake is one island", []string{"###", "..#", "###"}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Build a fresh grid per case: the function may mark it in place.
			if got := CountIslands(grid(c.rows...)); got != c.want {
				t.Errorf("CountIslands(%v) = %d, want %d", c.rows, got, c.want)
			}
		})
	}
}

func TestMaxNonAdjacentSum(t *testing.T) {
	cases := []struct {
		name string
		nums []int
		want int
	}{
		{"empty", nil, 0},
		{"single", []int{5}, 5},
		{"two picks the larger", []int{5, 9}, 9},
		{"skip one in the middle", []int{3, 2, 7, 10}, 13},
		{"skip two in a row", []int{2, 7, 9, 3, 1}, 12},
		{"big value carries", []int{5, 5, 10, 100, 10, 5}, 110},
		{"all equal", []int{4, 4, 4, 4, 4}, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MaxNonAdjacentSum(c.nums); got != c.want {
				t.Errorf("MaxNonAdjacentSum(%v) = %d, want %d", c.nums, got, c.want)
			}
		})
	}
}

func TestMaxNonAdjacentSumLinearTime(t *testing.T) {
	const n = 1_000_000
	nums := make([]int, n)
	for i := range nums {
		nums[i] = 1
	}
	start := time.Now()
	got := MaxNonAdjacentSum(nums)
	elapsed := time.Since(start)
	if got != n/2 {
		t.Fatalf("MaxNonAdjacentSum(%d ones) = %d, want %d", n, got, n/2)
	}
	if elapsed > timeGuard {
		t.Errorf("MaxNonAdjacentSum took %v on %d values — trying combinations is exponential; one take-or-skip decision per element is O(n)", elapsed, n)
	}
}

func TestMaxNonAdjacentSumConstantSpace(t *testing.T) {
	nums := make([]int, 100_000)
	for i := range nums {
		nums[i] = 1
	}
	allocs := testing.AllocsPerRun(10, func() {
		sinkInt = MaxNonAdjacentSum(nums)
	})
	if allocs > 0 {
		t.Errorf("MaxNonAdjacentSum allocated %.0f time(s) per run, want 0 — a full table is O(n) space; only the previous two results matter, so carry them in two variables", allocs)
	}
}
