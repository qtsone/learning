package memlab

import "testing"

func TestSummarize(t *testing.T) {
	cases := []struct {
		name string
		xs   []float64
		want Report
	}{
		{"nil", nil, Report{}},
		{"empty", []float64{}, Report{}},
		{"single", []float64{3.5}, Report{Min: 3.5, Max: 3.5, Sum: 3.5, N: 1}},
		{"mixed", []float64{3, -1, 2}, Report{Min: -1, Max: 3, Sum: 4, N: 3}},
		{"all negative", []float64{-5, -2, -9}, Report{Min: -9, Max: -2, Sum: -16, N: 3}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Summarize(c.xs); got != c.want {
				t.Errorf("Summarize(%v) = %+v, want %+v", c.xs, got, c.want)
			}
		})
	}
}

var sinkReport Report

func TestSummarizeDoesNotAllocate(t *testing.T) {
	xs := make([]float64, 1024)
	for i := range xs {
		xs[i] = float64(i%17) - 8
	}
	allocs := testing.AllocsPerRun(100, func() {
		sinkReport = Summarize(xs)
	})
	if allocs > 0 {
		t.Errorf("Summarize allocates %.0f time(s) per call, want 0 — return a value that stays on the stack instead of a pointer that escapes (and don't delegate to SummarizeHeap)", allocs)
	}
}
