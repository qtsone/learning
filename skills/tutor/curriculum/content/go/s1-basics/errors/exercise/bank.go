// Package bank is a tiny in-memory bank for practicing error handling.
package bank

import "errors"

// Sentinel errors: expected failures callers can detect with errors.Is.
var (
	ErrAccountNotFound   = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// Bank holds one balance per account name, in cents.
type Bank struct {
	balances map[string]int
}

// NewBank returns an empty bank, ready to use.
func NewBank() *Bank {
	return &Bank{balances: make(map[string]int)}
}

// Deposit adds amount cents to account, creating the account on its first
// deposit. A non-positive amount is rejected with an error and changes
// nothing.
func (b *Bank) Deposit(account string, amount int) error {
	// TODO: reject amount <= 0 with fmt.Errorf, then add to the balance.
	return nil
}

// Balance reports how many cents account holds. An unknown account yields an
// error wrapping ErrAccountNotFound that names the account.
func (b *Bank) Balance(account string) (int, error) {
	// TODO: comma-ok lookup; on a miss, wrap ErrAccountNotFound with %w.
	return 0, nil
}

// Withdraw removes amount cents from account. Failures (non-positive amount,
// unknown account, amount beyond the balance) leave balances untouched; see
// the acceptance criteria in LESSON.md for which error each case carries.
func (b *Bank) Withdraw(account string, amount int) error {
	// TODO: guard the amount, look up the balance, compare, then subtract.
	return nil
}

// Transfer moves amount cents from one account to the other. On failure it
// adds "transfer" context with both account names while keeping the
// underlying cause detectable via errors.Is.
func (b *Bank) Transfer(from, to string, amount int) error {
	// TODO: withdraw from `from`, deposit to `to`, wrapping any error with %w.
	return nil
}
