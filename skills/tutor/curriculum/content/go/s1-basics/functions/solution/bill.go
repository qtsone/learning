package main

import (
	"errors"
	"fmt"
)

// Sum returns the total of all prices, in cents.
// With no arguments it returns 0.
func Sum(prices ...int) int {
	total := 0
	for _, price := range prices {
		total += price
	}
	return total
}

// Split divides total cents evenly among people. It returns each person's
// share, the cents left over, and an error when people is less than 1.
func Split(total, people int) (share, remainder int, err error) {
	if people < 1 {
		return 0, 0, errors.New("split: need at least one person")
	}
	return total / people, total % people, nil
}

// SplitBill sums the prices and splits the total among people.
func SplitBill(people int, prices ...int) (share, remainder int, err error) {
	return Split(Sum(prices...), people)
}

// FormatCents renders cents as a dollar string: 1250 becomes "12.50".
func FormatCents(cents int) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
