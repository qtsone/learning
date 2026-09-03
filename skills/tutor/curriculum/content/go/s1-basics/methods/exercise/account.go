// Package bank is a tiny account ledger for practicing methods.
package bank

// Cents is an amount of money in whole cents. Amounts are never negative.
type Cents int

// Account is one customer's balance, in cents.
type Account struct {
	Owner   string
	Balance int
}

// CanAfford reports whether the account holds at least amount cents.
func (a Account) CanAfford(amount int) bool {
	// TODO: compare amount against the balance.
	return false
}

// Deposit adds amount cents to the balance. Negative amounts are ignored.
//
// BUG: this compiles and the body is correct, yet the deposit never
// reaches the caller. Fix the receiver, not the body.
func (a Account) Deposit(amount int) {
	if amount < 0 {
		return
	}
	a.Balance += amount
}

// Withdraw removes amount cents from the balance and reports success.
// A negative or unaffordable amount returns false and leaves the
// balance unchanged.
func (a *Account) Withdraw(amount int) bool {
	// TODO: guard, then subtract. CanAfford already knows the answer.
	return false
}

// Dollars renders the amount as dollars, like "$12.34" or "$0.05".
func (c Cents) Dollars() string {
	// TODO: split c into dollar and cent parts. fmt.Sprintf works like
	// fmt.Printf but returns the string instead of printing it, and the
	// verb %02d pads a number to two digits with leading zeros.
	return ""
}
