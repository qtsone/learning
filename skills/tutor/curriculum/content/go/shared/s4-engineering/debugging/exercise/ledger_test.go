package ledger

import (
	"math"
	"slices"
	"testing"
)

func closeEnough(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// day builds a deterministic day of n amounts: 0.25, 1.25, … 8.25, cycling.
func day(n int) []float64 {
	xs := make([]float64, n)
	for i := range xs {
		xs[i] = float64(i%9) + 0.25
	}
	return xs
}

func TestLowest(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"single amount", []float64{4.5}, 4.5},
		{"all purchases", []float64{12.5, 3.25, 7}, 3.25},
		{"includes a refund", []float64{12.5, -3, 7}, -3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Lowest(c.in); !closeEnough(got, c.want) {
				t.Errorf("Lowest(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestTotal(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"no amounts", nil, 0},
		{"one amount", []float64{2.5}, 2.5},
		{"four amounts", []float64{1.5, 2, 3, 4}, 10.5},
		{"five amounts", []float64{1.5, 2, 3, 4, 5}, 15.5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Total(c.in); !closeEnough(got, c.want) {
				t.Errorf("Total(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestTotalFullDay(t *testing.T) {
	in := day(103)
	var want float64
	for _, x := range in {
		want += x
	}
	if got := Total(in); !closeEnough(got, want) {
		t.Errorf("Total over %d amounts = %v, want %v — shrink the day until the failure disappears, then look at what the boundary has in common",
			len(in), got, want)
	}
}

func TestMedian(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"single amount", []float64{4.5}, 4.5},
		{"odd count", []float64{9, 1, 7}, 7},
		{"even count", []float64{9, 1, 7, 3}, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Median(c.in); !closeEnough(got, c.want) {
				t.Errorf("Median(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestMedianLeavesInputAlone(t *testing.T) {
	in := []float64{9, 1, 7, 3}
	before := slices.Clone(in)
	_ = Median(in)
	if !slices.Equal(in, before) {
		t.Errorf("Median reordered its input: before %v, after %v — callers rely on their slice staying put",
			before, in)
	}
}
