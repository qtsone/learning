package tdd

import (
	"fmt"
	"strconv"
	"strings"
)

// Add sums the non-negative integers in input, where "" sums to 0 and
// numbers are separated by commas or newlines. Negative numbers are
// rejected with an error naming every offender.
func Add(input string) (int, error) {
	if input == "" {
		return 0, nil
	}
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == '\n'
	})
	sum := 0
	var negatives []string
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return 0, fmt.Errorf("parse %q: %w", part, err)
		}
		if n < 0 {
			negatives = append(negatives, part)
			continue
		}
		sum += n
	}
	if len(negatives) > 0 {
		return 0, fmt.Errorf("negative numbers not allowed: %s", strings.Join(negatives, ", "))
	}
	return sum, nil
}
