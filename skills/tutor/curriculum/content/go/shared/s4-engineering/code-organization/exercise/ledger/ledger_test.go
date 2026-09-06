package ledger_test

import (
	"maps"
	"testing"

	"tutor.local/code-organization/expense"
	"tutor.local/code-organization/ledger"
)

func TestAddAndLen(t *testing.T) {
	var l ledger.Ledger
	if l.Len() != 0 {
		t.Fatalf("zero-value Ledger Len() = %d, want 0", l.Len())
	}
	l.Add(expense.Expense{Description: "coffee", Category: "food", AmountCents: 350})
	l.Add(expense.Expense{Description: "sandwich", Category: "food", AmountCents: 250})
	if got := l.Len(); got != 2 {
		t.Errorf("Len() after two Adds = %d, want 2", got)
	}
}

func TestTotal(t *testing.T) {
	var l ledger.Ledger
	l.Add(expense.Expense{Description: "coffee", Category: "food", AmountCents: 350})
	l.Add(expense.Expense{Description: "train ticket", Category: "travel", AmountCents: 9950})
	if got := l.Total(); got != 10300 {
		t.Errorf("Total() = %d, want 10300 (350 + 9950)", got)
	}
}

func TestTotalByCategory(t *testing.T) {
	var l ledger.Ledger
	l.Add(expense.Expense{Description: "coffee", Category: "food", AmountCents: 350})
	l.Add(expense.Expense{Description: "sandwich", Category: "food", AmountCents: 250})
	l.Add(expense.Expense{Description: "train ticket", Category: "travel", AmountCents: 9950})
	want := map[string]int{"food": 600, "travel": 9950}
	if got := l.TotalByCategory(); !maps.Equal(got, want) {
		t.Errorf("TotalByCategory() = %v, want %v", got, want)
	}
}

func TestEmptyLedger(t *testing.T) {
	var l ledger.Ledger
	if got := l.Total(); got != 0 {
		t.Errorf("empty Ledger Total() = %d, want 0", got)
	}
	if got := l.TotalByCategory(); len(got) != 0 {
		t.Errorf("empty Ledger TotalByCategory() = %v, want no entries", got)
	}
}
