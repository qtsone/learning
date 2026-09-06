package expense_test

import (
	"errors"
	"testing"

	"tutor.local/code-organization/expense"
)

func TestNewValid(t *testing.T) {
	e, err := expense.New("coffee", "food", 350)
	if err != nil {
		t.Fatalf("New(\"coffee\", \"food\", 350) returned error %v, want nil", err)
	}
	want := expense.Expense{Description: "coffee", Category: "food", AmountCents: 350}
	if e != want {
		t.Errorf("New(\"coffee\", \"food\", 350) = %+v, want %+v", e, want)
	}
}

func TestNewTrimsWhitespace(t *testing.T) {
	e, err := expense.New("  coffee ", " food ", 350)
	if err != nil {
		t.Fatalf("New with padded fields returned error %v, want nil", err)
	}
	if e.Description != "coffee" || e.Category != "food" {
		t.Errorf("New(%q, %q, 350) = %+v, want surrounding whitespace trimmed",
			"  coffee ", " food ", e)
	}
}

func TestNewRejectsInvalid(t *testing.T) {
	cases := []struct {
		name    string
		desc    string
		cat     string
		cents   int
		wantErr error
	}{
		{"empty description", "", "food", 100, expense.ErrEmptyDescription},
		{"blank description", "   ", "food", 100, expense.ErrEmptyDescription},
		{"empty category", "coffee", "", 100, expense.ErrEmptyCategory},
		{"blank category", "coffee", "  ", 100, expense.ErrEmptyCategory},
		{"zero amount", "coffee", "food", 0, expense.ErrNonPositiveAmount},
		{"negative amount", "coffee", "food", -50, expense.ErrNonPositiveAmount},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := expense.New(c.desc, c.cat, c.cents)
			if !errors.Is(err, c.wantErr) {
				t.Errorf("New(%q, %q, %d) error = %v, want %v",
					c.desc, c.cat, c.cents, err, c.wantErr)
			}
		})
	}
}
