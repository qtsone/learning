// Package report turns computed numbers into text for humans. It depends
// on plain data handed to it, not on the packages that produce the data.
package report

// FormatCents renders an amount in cents as dollars: 350 → "$3.50",
// 5 → "$0.05".
func FormatCents(cents int) string {
	// TODO
	return ""
}

// Summary renders per-category totals as one "category: amount" line per
// category, sorted alphabetically, followed by a "total:" line. Every
// line ends in a newline:
//
//	food: $6.00
//	travel: $99.50
//	total: $105.50
func Summary(byCategory map[string]int) string {
	// TODO: sort the category names, format each line with FormatCents,
	// then append the total line.
	return ""
}
