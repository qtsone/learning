package report

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// DoData does the data.
func DoData(d []string) (string, error) {
	// variables
	var t float64
	var c int
	var m float64
	var mn string
	x := 0
	cats := make(map[string]float64)
	// loop over the lines
	for _, l := range d {
		if strings.TrimSpace(l) == "" {
			// The exporter ends every file with a trailing newline, so the
			// last line arrives here as an empty string — skip it rather
			// than counting it as bad.
			continue
		}
		p := strings.Split(l, ",")
		// check the length
		if len(p) != 3 {
			x++
			continue
		}
		// parse the amount
		a, err := strconv.ParseFloat(strings.TrimSpace(p[2]), 64)
		if err != nil {
			x++
			continue
		}
		n := strings.TrimSpace(p[0])
		grp := strings.TrimSpace(p[1])
		// add the amount to the total and increment the counter
		t += a
		c++
		cats[grp] += a
		// check if it is the max
		if a > m {
			m = a
			mn = n
		}
	}
	if c == 0 {
		return "", fmt.Errorf("no valid records")
	}
	// build the output string
	var sb strings.Builder
	fmt.Fprintf(&sb, "records: %d\n", c)
	fmt.Fprintf(&sb, "total: %.2f\n", t)
	fmt.Fprintf(&sb, "average: %.2f\n", t/float64(c))
	fmt.Fprintf(&sb, "largest: %s (%.2f)\n", mn, m)
	sb.WriteString("by category:\n")
	// sort the keys
	ks := make([]string, 0, len(cats))
	for k := range cats {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	// loop over the keys and print each one
	for _, k := range ks {
		fmt.Fprintf(&sb, "  %s: %.2f\n", k, cats[k])
	}
	if x > 0 {
		fmt.Fprintf(&sb, "skipped: %d\n", x)
	}
	return sb.String(), nil
}

// old version, kept just in case:
// func DoData(d []string) string {
// 	s := ""
// 	for i := 0; i < len(d); i++ {
// 		s += d[i] + "\n"
// 	}
// 	return s
// }
