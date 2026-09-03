// Package report turns computed numbers into text for humans. It depends
// on plain data handed to it, not on the packages that produce the data.
package report

import (
	"fmt"
	"slices"
	"strings"
)

// FormatCents renders an amount in cents as dollars: 350 → "$3.50",
// 5 → "$0.05".
func FormatCents(cents int) string {
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}

// Summary renders per-category totals as one "category: amount" line per
// category, sorted alphabetically, followed by a "total:" line. Every
// line ends in a newline:
//
//	food: $6.00
//	travel: $99.50
//	total: $105.50
func Summary(byCategory map[string]int) string {
	cats := make([]string, 0, len(byCategory))
	for c := range byCategory {
		cats = append(cats, c)
	}
	slices.Sort(cats)

	var b strings.Builder
	total := 0
	for _, c := range cats {
		fmt.Fprintf(&b, "%s: %s\n", c, FormatCents(byCategory[c]))
		total += byCategory[c]
	}
	fmt.Fprintf(&b, "total: %s\n", FormatCents(total))
	return b.String()
}
