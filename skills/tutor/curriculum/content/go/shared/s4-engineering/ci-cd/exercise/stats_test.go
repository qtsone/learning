package stats

import (
	"math"
	"testing"
)

func TestMean(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"single value", []float64{5}, 5},
		{"three values", []float64{2, 4, 6}, 4},
		{"negatives cancel", []float64{-1, 1}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Mean(c.in)
			if err != nil {
				t.Fatalf("Mean(%v) returned error: %v", c.in, err)
			}
			if math.Abs(got-c.want) > 1e-9 {
				t.Errorf("Mean(%v) = %g, want %g", c.in, got, c.want)
			}
		})
	}
}

func TestMeanEmpty(t *testing.T) {
	if _, err := Mean(nil); err == nil {
		t.Error("Mean(nil) error = nil, want an error for empty input")
	}
}
