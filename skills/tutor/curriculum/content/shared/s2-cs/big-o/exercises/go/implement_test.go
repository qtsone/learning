package bigo

import (
	"testing"
	"time"
)

// Generous wall-clock guard: the required linear solutions finish in
// milliseconds; a quadratic one takes minutes on these input sizes.
const timeGuard = 3 * time.Second

func TestHasDuplicate(t *testing.T) {
	cases := []struct {
		name string
		xs   []int
		want bool
	}{
		{"empty", nil, false},
		{"single", []int{7}, false},
		{"no duplicates", []int{3, 1, 4, 5, 9}, false},
		{"adjacent duplicate", []int{1, 1, 2}, true},
		{"far-apart duplicate", []int{5, 9, 2, 7, 5}, true},
		{"negative duplicate", []int{-3, 0, -3}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HasDuplicate(c.xs); got != c.want {
				t.Errorf("HasDuplicate(%v) = %v, want %v", c.xs, got, c.want)
			}
		})
	}
}

func TestHasDuplicateLinearTime(t *testing.T) {
	const n = 300_000
	xs := make([]int, n)
	for i := range xs {
		xs[i] = i // all unique: the worst case, nothing to exit early on
	}
	start := time.Now()
	got := HasDuplicate(xs)
	elapsed := time.Since(start)
	if got {
		t.Fatalf("HasDuplicate on %d unique values = true, want false", n)
	}
	if elapsed > timeGuard {
		t.Errorf("HasDuplicate took %v on %d unique values — that growth smells O(n^2); make one pass with a map", elapsed, n)
	}
}

func TestHasDuplicateSorted(t *testing.T) {
	cases := []struct {
		name string
		xs   []int
		want bool
	}{
		{"empty", nil, false},
		{"single", []int{4}, false},
		{"sorted no duplicates", []int{-2, 0, 3, 7, 11}, false},
		{"duplicate in middle", []int{1, 2, 2, 5}, true},
		{"duplicate at end", []int{1, 3, 8, 8}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HasDuplicateSorted(c.xs); got != c.want {
				t.Errorf("HasDuplicateSorted(%v) = %v, want %v", c.xs, got, c.want)
			}
		})
	}
}

var sink bool

func TestHasDuplicateSortedConstantSpace(t *testing.T) {
	const n = 100_000
	xs := make([]int, n)
	for i := range xs {
		xs[i] = i
	}
	allocs := testing.AllocsPerRun(10, func() {
		sink = HasDuplicateSorted(xs)
	})
	if allocs > 0 {
		t.Errorf("HasDuplicateSorted allocated %.0f time(s) per run, want 0 — O(1) auxiliary space means no maps and no new slices; compare neighbors", allocs)
	}
}

func TestCountCommon(t *testing.T) {
	cases := []struct {
		name string
		xs   []int
		ys   []int
		want int
	}{
		{"both empty", nil, nil, 0},
		{"one empty", []int{1, 2}, nil, 0},
		{"disjoint", []int{1, 2, 3}, []int{4, 5}, 0},
		{"some common", []int{1, 2, 3, 4}, []int{3, 4, 5}, 2},
		{"duplicates do not inflate", []int{1, 1, 2}, []int{1, 1, 3}, 1},
		{"all common", []int{7, 8}, []int{8, 7}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CountCommon(c.xs, c.ys); got != c.want {
				t.Errorf("CountCommon(%v, %v) = %d, want %d", c.xs, c.ys, got, c.want)
			}
		})
	}
}

func TestCountCommonLinearTime(t *testing.T) {
	const n = 200_000
	xs := make([]int, n)
	ys := make([]int, n)
	for i := 0; i < n; i++ {
		xs[i] = 2 * i   // evens
		ys[i] = 2*i + 1 // odds: no overlap, no early exit possible
	}
	start := time.Now()
	got := CountCommon(xs, ys)
	elapsed := time.Since(start)
	if got != 0 {
		t.Fatalf("CountCommon(evens, odds) = %d, want 0", got)
	}
	if elapsed > timeGuard {
		t.Errorf("CountCommon took %v on two %d-element slices — nested loops are O(n*m); build a set from one slice and scan the other", elapsed, n)
	}
}
