// Package ledger stores validated expenses and answers questions about
// them. It may import expense (the layer below) and nothing above it.
package ledger

import "tutor.local/code-organization/expense"

// Ledger collects expenses. Its zero value is ready to use.
type Ledger struct {
	items []expense.Expense
}

// Add stores one expense.
func (l *Ledger) Add(e expense.Expense) {
	l.items = append(l.items, e)
}

// Len reports how many expenses are stored.
func (l *Ledger) Len() int {
	return len(l.items)
}

// Total returns the sum of all amounts, in cents.
func (l *Ledger) Total() int {
	total := 0
	for _, e := range l.items {
		total += e.AmountCents
	}
	return total
}

// TotalByCategory returns the per-category sums, in cents.
func (l *Ledger) TotalByCategory() map[string]int {
	totals := make(map[string]int)
	for _, e := range l.items {
		totals[e.Category] += e.AmountCents
	}
	return totals
}
