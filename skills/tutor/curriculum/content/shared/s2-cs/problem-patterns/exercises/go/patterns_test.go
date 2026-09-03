package patterns

import (
	"strings"
	"testing"
	"time"
)

// Generous wall-clock guard: the required linear solutions finish in
// milliseconds; a quadratic one takes minutes on these input sizes.
const timeGuard = 3 * time.Second

// Package-level sinks keep the compiler from optimizing calls away in the
// allocation probes.
var (
	sinkInt  int
	sinkBool bool
)

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPairSum(t *testing.T) {
	cases := []struct {
		name   string
		nums   []int
		target int
		wantOK bool
	}{
		{"empty", nil, 5, false},
		{"single element cannot pair", []int{5}, 10, false},
		{"pair at both ends", []int{1, 2, 3, 9}, 10, true},
		{"pair in the middle", []int{1, 3, 4, 6, 10}, 7, true},
		{"equal elements", []int{3, 3}, 6, true},
		{"negatives", []int{-4, -1, 2, 6}, 1, true},
		{"no pair", []int{1, 2, 4, 4}, 100, false},
		{"target below every pair", []int{2, 4, 6}, 3, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			i, j, ok := PairSum(c.nums, c.target)
			if ok != c.wantOK {
				t.Fatalf("PairSum(%v, %d) ok = %v, want %v", c.nums, c.target, ok, c.wantOK)
			}
			if !ok {
				return
			}
			if i < 0 || j >= len(c.nums) || i >= j {
				t.Fatalf("PairSum(%v, %d) = (%d, %d): want indexes inside the slice with i < j", c.nums, c.target, i, j)
			}
			if sum := c.nums[i] + c.nums[j]; sum != c.target {
				t.Errorf("PairSum(%v, %d) = (%d, %d), but nums[%d]+nums[%d] = %d, want %d", c.nums, c.target, i, j, i, j, sum, c.target)
			}
		})
	}
}

func TestPairSumLinearTime(t *testing.T) {
	const n = 400_000
	nums := make([]int, n)
	for i := range nums {
		nums[i] = 2 * i // all even: an odd target has no answer and no early exit
	}
	start := time.Now()
	_, _, ok := PairSum(nums, 999_999)
	elapsed := time.Since(start)
	if ok {
		t.Fatalf("PairSum(evens, odd target) found a pair, want none")
	}
	if elapsed > timeGuard {
		t.Errorf("PairSum took %v on %d sorted values — checking every pair is O(n^2); converge one pointer from each end", elapsed, n)
	}
}

func TestPairSumConstantSpace(t *testing.T) {
	const n = 200_000
	nums := make([]int, n)
	for i := range nums {
		nums[i] = 2 * i
	}
	allocs := testing.AllocsPerRun(10, func() {
		sinkInt, _, sinkBool = PairSum(nums, 999_999)
	})
	if allocs > 0 {
		t.Errorf("PairSum allocated %.0f time(s) per run, want 0 — the input is already sorted, so no map is needed; two pointers use O(1) space", allocs)
	}
}

func TestRemoveDuplicates(t *testing.T) {
	cases := []struct {
		name string
		nums []int
		want []int
	}{
		{"empty", nil, nil},
		{"single", []int{7}, []int{7}},
		{"already unique", []int{1, 2, 3}, []int{1, 2, 3}},
		{"all the same", []int{4, 4, 4, 4}, []int{4}},
		{"mixed runs", []int{1, 1, 2, 3, 3, 3, 9}, []int{1, 2, 3, 9}},
		{"negatives", []int{-3, -3, -1, 0, 0}, []int{-3, -1, 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			nums := append([]int(nil), c.nums...)
			k := RemoveDuplicates(nums)
			if k != len(c.want) {
				t.Fatalf("RemoveDuplicates(%v) = %d, want %d", c.nums, k, len(c.want))
			}
			if !equalInts(nums[:k], c.want) {
				t.Errorf("RemoveDuplicates(%v): first %d elements = %v, want %v", c.nums, k, nums[:k], c.want)
			}
		})
	}
}

func TestRemoveDuplicatesLinearTime(t *testing.T) {
	const n = 600_000
	nums := make([]int, n)
	for i := range nums {
		nums[i] = i / 2 // every value appears exactly twice
	}
	start := time.Now()
	k := RemoveDuplicates(nums)
	elapsed := time.Since(start)
	if k != n/2 {
		t.Fatalf("RemoveDuplicates on %d values (each doubled) = %d, want %d", n, k, n/2)
	}
	for i := 0; i < k; i++ {
		if nums[i] != i {
			t.Fatalf("nums[%d] = %d after compaction, want %d", i, nums[i], i)
		}
	}
	if elapsed > timeGuard {
		t.Errorf("RemoveDuplicates took %v on %d values — shifting the tail for every duplicate is O(n^2); one pass with a read and a write index", elapsed, n)
	}
}

func TestRemoveDuplicatesConstantSpace(t *testing.T) {
	nums := make([]int, 100_000)
	for i := range nums {
		nums[i] = i / 2
	}
	allocs := testing.AllocsPerRun(10, func() {
		sinkInt = RemoveDuplicates(nums)
	})
	if allocs > 0 {
		t.Errorf("RemoveDuplicates allocated %.0f time(s) per run, want 0 — in place means no second slice and no map; overwrite the prefix", allocs)
	}
}

func TestMaxWindowSum(t *testing.T) {
	cases := []struct {
		name    string
		nums    []int
		k       int
		wantSum int
		wantOK  bool
	}{
		{"k zero", []int{1, 2}, 0, 0, false},
		{"k negative", []int{1, 2}, -3, 0, false},
		{"k larger than slice", []int{1, 2}, 3, 0, false},
		{"empty slice", nil, 1, 0, false},
		{"k equals length", []int{2, 4, 6}, 3, 12, true},
		{"k one picks the max element", []int{5, 1, 9, 3}, 1, 9, true},
		{"classic", []int{1, 4, 2, 10, 23, 3, 1, 0, 20}, 4, 39, true},
		{"all negative", []int{-2, -5, -1, -4}, 2, -5, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sum, ok := MaxWindowSum(c.nums, c.k)
			if sum != c.wantSum || ok != c.wantOK {
				t.Errorf("MaxWindowSum(%v, %d) = (%d, %v), want (%d, %v)", c.nums, c.k, sum, ok, c.wantSum, c.wantOK)
			}
		})
	}
}

func TestMaxWindowSumLinearTime(t *testing.T) {
	const n, k = 240_000, 120_000
	nums := make([]int, n)
	for i := range nums {
		nums[i] = 1
	}
	start := time.Now()
	sum, ok := MaxWindowSum(nums, k)
	elapsed := time.Since(start)
	if !ok || sum != k {
		t.Fatalf("MaxWindowSum(%d ones, k=%d) = (%d, %v), want (%d, true)", n, k, sum, ok, k)
	}
	if elapsed > timeGuard {
		t.Errorf("MaxWindowSum took %v with n=%d, k=%d — summing every window from scratch is O(n*k); slide the sum: add the entering element, subtract the leaving one", elapsed, n, k)
	}
}

func TestLongestUniqueRun(t *testing.T) {
	cases := []struct {
		name string
		s    string
		want int
	}{
		{"empty", "", 0},
		{"single rune", "x", 1},
		{"all distinct", "abcdef", 6},
		{"classic abcabcbb", "abcabcbb", 3},
		{"all same", "bbbbb", 1},
		{"pwwkew", "pwwkew", 3},
		{"window must not reopen", "abba", 2},
		{"multibyte runes count once", "día", 3},
		{"CJK", "日本語日本語", 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := LongestUniqueRun(c.s); got != c.want {
				t.Errorf("LongestUniqueRun(%q) = %d, want %d", c.s, got, c.want)
			}
		})
	}
}

func TestLongestUniqueRunLinearTime(t *testing.T) {
	const n = 300_000
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteRune(rune(0x20000 + i)) // n distinct runes, all valid
	}
	start := time.Now()
	got := LongestUniqueRun(b.String())
	elapsed := time.Since(start)
	if got != n {
		t.Fatalf("LongestUniqueRun(%d distinct runes) = %d, want %d", n, got, n)
	}
	if elapsed > timeGuard {
		t.Errorf("LongestUniqueRun took %v on %d distinct runes — rescanning from every start is O(n^2); slide a window and remember each rune's most recent index", elapsed, n)
	}
}
