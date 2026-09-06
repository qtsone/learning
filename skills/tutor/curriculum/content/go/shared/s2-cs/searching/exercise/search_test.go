package search

import "testing"

func TestLinearSearch(t *testing.T) {
	cases := []struct {
		name       string
		xs         []int
		target     int
		wantIndex  int
		wantProbes int
	}{
		{"found at front", []int{7, 3, 9}, 7, 0, 1},
		{"found in middle", []int{7, 3, 9}, 3, 1, 2},
		{"found at back", []int{7, 3, 9}, 9, 2, 3},
		{"absent", []int{7, 3, 9}, 4, -1, 3},
		{"empty slice", []int{}, 1, -1, 0},
		{"unsorted input is fine", []int{5, 1, 4, 1, 5}, 4, 2, 3},
		{"first match wins", []int{2, 8, 2}, 2, 0, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			index, probes := LinearSearch(c.xs, c.target)
			if index != c.wantIndex {
				t.Errorf("LinearSearch(%v, %d) index = %d, want %d",
					c.xs, c.target, index, c.wantIndex)
			}
			if probes != c.wantProbes {
				t.Errorf("LinearSearch(%v, %d) probes = %d, want %d (count every element examined; stop at the first match)",
					c.xs, c.target, probes, c.wantProbes)
			}
		})
	}
}

func TestBinarySearch(t *testing.T) {
	cases := []struct {
		name   string
		xs     []int
		target int
		found  bool
	}{
		{"empty slice", []int{}, 3, false},
		{"single element hit", []int{5}, 5, true},
		{"single element miss", []int{5}, 3, false},
		{"first element", []int{1, 3, 5, 7, 9}, 1, true},
		{"last element", []int{1, 3, 5, 7, 9}, 9, true},
		{"middle element", []int{1, 3, 5, 7, 9}, 5, true},
		{"even length hit", []int{1, 3, 5, 7}, 7, true},
		{"absent between elements", []int{1, 3, 5, 7, 9}, 4, false},
		{"below all elements", []int{1, 3, 5, 7, 9}, 0, false},
		{"above all elements", []int{1, 3, 5, 7, 9}, 10, false},
		{"duplicates: any matching index", []int{1, 2, 2, 2, 3}, 2, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			index, _ := BinarySearch(c.xs, c.target)
			if !c.found {
				if index != -1 {
					t.Errorf("BinarySearch(%v, %d) index = %d, want -1 for an absent target",
						c.xs, c.target, index)
				}
				return
			}
			if index < 0 || index >= len(c.xs) || c.xs[index] != c.target {
				t.Errorf("BinarySearch(%v, %d) index = %d, want any index holding %d",
					c.xs, c.target, index, c.target)
			}
		})
	}
	t.Run("empty slice costs zero probes", func(t *testing.T) {
		if _, probes := BinarySearch(nil, 3); probes != 0 {
			t.Errorf("BinarySearch(nil, 3) probes = %d, want 0 (nothing to examine)", probes)
		}
	})
}

func TestFirstAndLastOccurrence(t *testing.T) {
	cases := []struct {
		name      string
		xs        []int
		target    int
		wantFirst int
		wantLast  int
	}{
		{"unique element", []int{1, 3, 5}, 3, 1, 1},
		{"run in the middle", []int{1, 2, 2, 2, 3}, 2, 1, 3},
		{"run at the front", []int{2, 2, 2, 5}, 2, 0, 2},
		{"run at the back", []int{1, 4, 4}, 4, 1, 2},
		{"whole slice equal", []int{6, 6, 6, 6}, 6, 0, 3},
		{"absent above all", []int{1, 2, 2, 3}, 5, -1, -1},
		{"absent below all", []int{1, 2, 2, 3}, 0, -1, -1},
		{"absent between elements", []int{1, 3, 3, 9}, 2, -1, -1},
		{"empty slice", []int{}, 1, -1, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if first, _ := FirstOccurrence(c.xs, c.target); first != c.wantFirst {
				t.Errorf("FirstOccurrence(%v, %d) = %d, want %d",
					c.xs, c.target, first, c.wantFirst)
			}
			if last, _ := LastOccurrence(c.xs, c.target); last != c.wantLast {
				t.Errorf("LastOccurrence(%v, %d) = %d, want %d",
					c.xs, c.target, last, c.wantLast)
			}
		})
	}
}

// bigN is ~a million elements: log2(bigN) = 20, so honest halving examines
// about 21 elements. probeBudget leaves generous headroom for counting-style
// differences while staying five orders of magnitude below a linear scan.
const (
	bigN        = 1 << 20
	probeBudget = 64
)

func bigSorted() []int {
	xs := make([]int, bigN)
	for i := range xs {
		xs[i] = 2 * i // even values, so every odd target is absent
	}
	return xs
}

func TestBinarySearchHalvesTheInterval(t *testing.T) {
	xs := bigSorted()
	for _, target := range []int{0, 2 * (bigN - 1), 2*12345 + 1, -1, 2 * bigN} {
		if _, probes := BinarySearch(xs, target); probes > probeBudget {
			t.Errorf("BinarySearch over %d elements examined %d of them for target %d; halving needs ~21 — does every iteration shrink the interval?",
				bigN, probes, target)
		}
	}
	if index, _ := BinarySearch(xs, 2*12345); index != 12345 {
		t.Errorf("BinarySearch(bigSorted(), %d) index = %d, want 12345", 2*12345, index)
	}
}

func TestLinearVsBinaryCostGap(t *testing.T) {
	xs := bigSorted()
	const absent = 1
	if _, probes := LinearSearch(xs, absent); probes != bigN {
		t.Errorf("LinearSearch miss over %d elements probes = %d, want %d (a miss must examine everything)",
			bigN, probes, bigN)
	}
	if _, probes := BinarySearch(xs, absent); probes > probeBudget {
		t.Errorf("BinarySearch miss over %d elements examined %d; sortedness should make a miss as cheap as a hit (~21 probes)",
			bigN, probes)
	}
}

func TestOccurrenceSearchesHalveTheInterval(t *testing.T) {
	xs := make([]int, bigN)
	for i := range xs {
		xs[i] = 7
	}
	first, fp := FirstOccurrence(xs, 7)
	if first != 0 {
		t.Errorf("FirstOccurrence(all-7s, 7) = %d, want 0", first)
	}
	if fp > probeBudget {
		t.Errorf("FirstOccurrence examined %d of %d equal elements; walking sideways from a match is O(n) — keep halving to the boundary instead",
			fp, bigN)
	}
	last, lp := LastOccurrence(xs, 7)
	if last != bigN-1 {
		t.Errorf("LastOccurrence(all-7s, 7) = %d, want %d", last, bigN-1)
	}
	if lp > probeBudget {
		t.Errorf("LastOccurrence examined %d of %d equal elements; walking sideways from a match is O(n) — keep halving to the boundary instead",
			lp, bigN)
	}
}
