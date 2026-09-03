package main

import "fmt"

// Sum returns the total of all prices, in cents.
// With no arguments it returns 0.
func Sum(prices ...int) int {
	// TODO: add every price with a for … range loop and return the total.
	return 0
}

// Split divides total cents evenly among people. It returns each person's
// share, the cents left over, and an error when people is less than 1.
func Split(total, people int) (share, remainder int, err error) {
	// TODO: when people < 1, return zero results and an error built with
	// errors.New (you'll need to import "errors").
	// TODO: otherwise compute share and remainder with / and %.
	return 0, 0, nil
}

// SplitBill sums the prices and splits the total among people.
func SplitBill(people int, prices ...int) (share, remainder int, err error) {
	// TODO: build this from Sum and Split — don't re-add the prices here.
	// Passing a slice (or a variadic parameter) onward needs the spread:
	// Sum(prices...).
	return 0, 0, nil
}

// FormatCents renders cents as a dollar string: 1250 becomes "12.50".
// Already done — read it as an example of a small helper with one job.
func FormatCents(cents int) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
