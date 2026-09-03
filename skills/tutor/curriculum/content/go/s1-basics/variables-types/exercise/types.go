package main

// Days of the week, numbered 1-7 like on a calendar.
// TODO: rewrite this block so a single iota expression on the first line
// produces Monday = 1 through Sunday = 7.
const (
	Monday    = 0
	Tuesday   = 0
	Wednesday = 0
	Thursday  = 0
	Friday    = 0
	Saturday  = 0
	Sunday    = 0
)

// ZeroReport declares one variable of each basic type without assigning a
// value and reports what Go initialized them to, in the form:
//
//	count=0 price=0 name="" active=false
func ZeroReport() string {
	// TODO: declare count (int), price (float64), name (string), and
	// active (bool) with var and NO explicit value, then format them with
	// fmt.Sprintf using %d, %v, %q, and %t.
	return ""
}

// Average returns sum divided by count, keeping the fraction.
func Average(sum int, count int) float64 {
	// TODO: 7 divided by 2 must give 3.5, not 3 — think about *when*
	// you convert.
	return 0
}

// PriceTag formats a price given in cents as a label like "coffee: $3.50".
func PriceTag(item string, cents int) string {
	// TODO: convert cents to dollars (use := for the in-between value),
	// then format with fmt.Sprintf — %.2f keeps exactly two decimals.
	return ""
}
