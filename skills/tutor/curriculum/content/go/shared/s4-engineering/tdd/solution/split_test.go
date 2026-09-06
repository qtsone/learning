package tdd

import (
	"slices"
	"testing"
)

// This is the test Part C asks you to write BEFORE fixing SplitEvenly.
// On the buggy starter, "remainder goes to the first shares" fails:
// SplitEvenly(100, 3) returns [33 33 33], which loses a cent.
func TestSplitEvenly(t *testing.T) {
	cases := []struct {
		name  string
		total int
		n     int
		want  []int
	}{
		{"even split", 100, 4, []int{25, 25, 25, 25}},
		{"remainder goes to the first shares", 100, 3, []int{34, 33, 33}},
		{"single share", 7, 1, []int{7}},
		{"more shares than cents", 2, 3, []int{1, 1, 0}},
		{"zero total", 0, 2, []int{0, 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SplitEvenly(c.total, c.n)
			if !slices.Equal(got, c.want) {
				t.Errorf("SplitEvenly(%d, %d) = %v, want %v", c.total, c.n, got, c.want)
			}
			sum := 0
			for _, s := range got {
				sum += s
			}
			if sum != c.total {
				t.Errorf("shares sum to %d, want %d — money must not appear or vanish", sum, c.total)
			}
		})
	}
}
