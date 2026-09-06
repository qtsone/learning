package main

import (
	"fmt"
	"os"

	"tutor.local/code-organization/expense"
	"tutor.local/code-organization/ledger"
	"tutor.local/code-organization/report"
)

func main() {
	raw := []struct {
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

	var l ledger.Ledger
	for _, r := range raw {
		e, err := expense.New(r.desc, r.cat, r.cents)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %q: %v\n", r.desc, err)
			continue
		}
		l.Add(e)
	}
	fmt.Print(report.Summary(l.TotalByCategory()))
}
