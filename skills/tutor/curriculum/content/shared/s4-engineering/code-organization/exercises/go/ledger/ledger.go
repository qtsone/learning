// Package ledger stores validated expenses and answers questions about
// them. It may import expense (the layer below) and nothing above it.
package ledger

import (
	"tutor.local/code-organization/expense"
	"tutor.local/code-organization/report"
)

// Ledger collects expenses. Its zero value is ready to use.
type Ledger struct {
	items []expense.Expense
}

// Add stores one expense.
func (l *Ledger) Add(e expense.Expense) {
	// TODO: append e to l.items.
}

// Len reports how many expenses are stored.
func (l *Ledger) Len() int {
	// TODO
	return 0
}

// Total returns the sum of all amounts, in cents.
func (l *Ledger) Total() int {
	// TODO
	return 0
}

// TotalByCategory returns the per-category sums, in cents.
func (l *Ledger) TotalByCategory() map[string]int {
	// TODO
	return nil
}

// Describe returns a display line for the ledger's total.
// TODO: this is presentation logic, and it drags in the report package —
// a dependency pointing the wrong way (TestDependencyDirection fails
// because of it). Decide where this behavior belongs and remove it here.
func (l *Ledger) Describe() string {
	return "total: " + report.FormatCents(l.Total())
}
