// Package stats provides tiny numeric summaries.
package stats

import "fmt"

// Mean returns the arithmetic mean of values.
// It returns an error when values is empty.
func Mean(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("stats: mean of empty input")
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values)-1), nil
}

// Describe returns a one-line summary such as "n=3 mean=4".
// Empty input yields "n=0".
func Describe(values []float64) string {
	m, err := Mean(values)
	if err != nil {
		return "n=0"
	}
	return fmt.Sprintf("n=%s mean=%g", len(values), m)
}
