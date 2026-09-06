// Package expense defines the expense record and its validation rules.
// It is the bottom of this project: every other package may import it,
// so it must import nothing else from this module.
package expense

import (
	"errors"
	"strings"
)

var (
	ErrEmptyDescription  = errors.New("expense: empty description")
	ErrEmptyCategory     = errors.New("expense: empty category")
	ErrNonPositiveAmount = errors.New("expense: amount must be positive")
)

// Expense is one validated spend, with the amount in cents.
type Expense struct {
	Description string
	Category    string
	AmountCents int
}

// New validates the fields and returns the Expense. Description and
// category are trimmed of surrounding whitespace; blank ones and
// non-positive amounts are rejected with the sentinel errors above.
func New(description, category string, amountCents int) (Expense, error) {
	description = strings.TrimSpace(description)
	category = strings.TrimSpace(category)
	switch {
	case description == "":
		return Expense{}, ErrEmptyDescription
	case category == "":
		return Expense{}, ErrEmptyCategory
	case amountCents <= 0:
		return Expense{}, ErrNonPositiveAmount
	}
	return Expense{Description: description, Category: category, AmountCents: amountCents}, nil
}
