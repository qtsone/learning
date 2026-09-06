package main

import (
	"fmt"
	"os"
	"sort"
)

// This program works — run it. Everything lives in one blob: data,
// validation rules, arithmetic, and money formatting, all interleaved.
// Your job (LESSON.md): move each concern into the package built for it
// (expense, ledger, report) and shrink main to wiring.
func main() {
	items := []struct {
		desc  string
		cat   string
		cents int
	}{
		{"coffee", "food", 350},
		{"sandwich", "food", 250},
		{"train ticket", "travel", 9950},
		{"", "food", 400},
		{"pens", "office", -200},
	}

	totals := map[string]int{}
	for _, it := range items {
		if it.desc == "" || it.cat == "" || it.cents <= 0 {
			fmt.Fprintln(os.Stderr, "skipping invalid expense:", it.desc)
			continue
		}
		totals[it.cat] += it.cents
	}

	cats := make([]string, 0, len(totals))
	total := 0
	for c, n := range totals {
		cats = append(cats, c)
		total += n
	}
	sort.Strings(cats)
	for _, c := range cats {
		fmt.Printf("%s: $%d.%02d\n", c, totals[c]/100, totals[c]%100)
	}
	fmt.Printf("total: $%d.%02d\n", total/100, total%100)
}
