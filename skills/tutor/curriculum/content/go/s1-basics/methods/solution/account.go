// Package bank is a tiny account ledger for practicing methods.
package bank

import "fmt"

// Cents is an amount of money in whole cents. Amounts are never negative.
type Cents int

// Account is one customer's balance, in cents.
type Account struct {
	Owner   string
	Balance int
}

// CanAfford reports whether the account holds at least amount cents.
func (a Account) CanAfford(amount int) bool {
	return amount <= a.Balance
}

// Deposit adds amount cents to the balance. Negative amounts are ignored.
func (a *Account) Deposit(amount int) {
	if amount < 0 {
		return
	}
	a.Balance += amount
}

// Withdraw removes amount cents from the balance and reports success.
// A negative or unaffordable amount returns false and leaves the
// balance unchanged.
func (a *Account) Withdraw(amount int) bool {
	if amount < 0 || !a.CanAfford(amount) {
		return false
	}
	a.Balance -= amount
	return true
}

// Dollars renders the amount as dollars, like "$12.34" or "$0.05".
func (c Cents) Dollars() string {
	return fmt.Sprintf("$%d.%02d", c/100, c%100)
}
