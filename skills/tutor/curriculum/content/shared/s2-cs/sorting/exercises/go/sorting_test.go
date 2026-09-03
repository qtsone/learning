package sorting

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"
)

func TestInsertionSort(t *testing.T) {
	cases := []struct {
		name string
		in   []int
		want []int
	}{
		{"empty", []int{}, []int{}},
		{"single", []int{7}, []int{7}},
		{"already sorted", []int{1, 2, 3, 4}, []int{1, 2, 3, 4}},
		{"reverse", []int{4, 3, 2, 1}, []int{1, 2, 3, 4}},
		{"duplicates", []int{3, 1, 3, 1}, []int{1, 1, 3, 3}},
		{"negatives", []int{0, -5, 12, -5, 7}, []int{-5, -5, 0, 7, 12}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := slices.Clone(c.in)
			InsertionSort(got)
			if !slices.Equal(got, c.want) {
				t.Errorf("InsertionSort(%v) left %v, want %v", c.in, got, c.want)
			}
		})
	}
}

// Complexity probe: insertion sort's best case is O(n) — on sorted input the
// inner loop must stop immediately. A non-adaptive quadratic pass over
// 200,000 elements blows far past this generous deadline.
func TestInsertionSortIsLinearOnSortedInput(t *testing.T) {
	const n = 200_000
	nums := make([]int, n)
	for i := range nums {
		nums[i] = i
	}
	start := time.Now()
	InsertionSort(nums)
	elapsed := time.Since(start)
	if !slices.IsSorted(nums) {
		t.Fatal("InsertionSort corrupted an already-sorted input")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("InsertionSort took %v on an already-sorted slice of %d; "+
			"it should be O(n) there — make the inner loop stop at the first "+
			"element that is not bigger than the key", elapsed, n)
	}
}

func TestMerge(t *testing.T) {
	cases := []struct {
		name string
		a, b []int
		want []int
	}{
		{"both empty", nil, nil, []int{}},
		{"left empty", nil, []int{1, 2}, []int{1, 2}},
		{"right empty", []int{1, 2}, nil, []int{1, 2}},
		{"interleaved", []int{1, 4, 6}, []int{2, 3, 5}, []int{1, 2, 3, 4, 5, 6}},
		{"duplicates across both", []int{1, 3, 3}, []int{2, 3}, []int{1, 2, 3, 3, 3}},
		{"one exhausts first", []int{10, 20}, []int{1, 2, 3}, []int{1, 2, 3, 10, 20}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Merge(c.a, c.b); !slices.Equal(got, c.want) {
				t.Errorf("Merge(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestMergeSort(t *testing.T) {
	cases := []struct {
		name string
		in   []int
		want []int
	}{
		{"empty", []int{}, []int{}},
		{"single", []int{7}, []int{7}},
		{"two out of order", []int{2, 1}, []int{1, 2}},
		{"reverse", []int{5, 4, 3, 2, 1}, []int{1, 2, 3, 4, 5}},
		{"duplicates", []int{2, 3, 2, 1, 3}, []int{1, 2, 2, 3, 3}},
		{"negatives", []int{0, -5, 12, -5, 7}, []int{-5, -5, 0, 7, 12}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MergeSort(c.in); !slices.Equal(got, c.want) {
				t.Errorf("MergeSort(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestMergeSortLeavesInputUntouched(t *testing.T) {
	in := []int{3, 1, 2}
	MergeSort(in)
	if !slices.Equal(in, []int{3, 1, 2}) {
		t.Fatalf("MergeSort modified its input (now %v) — return a new slice "+
			"instead of sorting in place", in)
	}
}

// Complexity probe: 500,000 elements is trivial for O(n log n) (well under a
// second) and hopeless for O(n^2) (hundreds of seconds).
func TestMergeSortHandlesLargeInput(t *testing.T) {
	const n = 500_000
	r := rand.New(rand.NewPCG(1, 2))
	nums := make([]int, n)
	for i := range nums {
		nums[i] = r.IntN(1_000_000)
	}
	want := slices.Clone(nums)
	slices.Sort(want)

	start := time.Now()
	got := MergeSort(nums)
	elapsed := time.Since(start)

	if !slices.Equal(got, want) {
		t.Fatalf("MergeSort on %d elements did not produce the sorted "+
			"permutation of its input (len=%d, want %d)", n, len(got), n)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("MergeSort took %v on %d elements; O(n log n) finishes well "+
			"under a second — is an O(n^2) step hiding inside?", elapsed, n)
	}
}

func TestRankPlayers(t *testing.T) {
	players := []Player{
		{"dana", 90},
		{"alice", 120},
		{"bob", 90},
		{"carol", 150},
	}
	RankPlayers(players)
	want := []Player{
		{"carol", 150},
		{"alice", 120},
		{"bob", 90},
		{"dana", 90},
	}
	if !slices.Equal(players, want) {
		t.Errorf("RankPlayers = %v, want %v (Score descending, ties by Name ascending)",
			players, want)
	}
}

func TestSortByScoreKeepsEqualScoresInInputOrder(t *testing.T) {
	players := []Player{
		{"zoe", 100},
		{"adam", 250},
		{"mia", 100},
		{"liam", 100},
		{"ivy", 250},
	}
	SortByScore(players)
	want := []Player{
		{"adam", 250},
		{"ivy", 250},
		{"zoe", 100},
		{"mia", 100},
		{"liam", 100},
	}
	if !slices.Equal(players, want) {
		t.Errorf("SortByScore = %v, want %v (equal scores keep input order)",
			players, want)
	}
}

// Stability probe at scale: names encode each player's original position, so
// within any group of equal scores the names must still be in ascending
// order. Unstable sorts shuffle equals once inputs get big enough.
func TestSortByScoreIsStableAtScale(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	players := make([]Player, 1000)
	for i := range players {
		players[i] = Player{Name: fmt.Sprintf("p%04d", i), Score: r.IntN(3)}
	}
	SortByScore(players)
	for i := 1; i < len(players); i++ {
		prev, cur := players[i-1], players[i]
		if prev.Score < cur.Score {
			t.Fatalf("not sorted by Score descending at index %d: %v before %v",
				i, prev, cur)
		}
		if prev.Score == cur.Score && prev.Name > cur.Name {
			t.Fatalf("unstable: %v now precedes %v although %v came first in "+
				"the input — equal scores must keep their original order",
				prev, cur, cur)
		}
	}
}
