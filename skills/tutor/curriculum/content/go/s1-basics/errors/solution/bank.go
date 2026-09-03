// Package bank is a tiny in-memory bank for practicing error handling.
package bank

import (
	"errors"
	"fmt"
)

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
	if amount <= 0 {
		return fmt.Errorf("deposit %d to %q: amount must be positive", amount, account)
	}
	b.balances[account] += amount
	return nil
}

// Balance reports how many cents account holds. An unknown account yields an
// error wrapping ErrAccountNotFound that names the account.
func (b *Bank) Balance(account string) (int, error) {
	balance, ok := b.balances[account]
	if !ok {
		return 0, fmt.Errorf("balance of %q: %w", account, ErrAccountNotFound)
	}
	return balance, nil
}

// Withdraw removes amount cents from account. Failures (non-positive amount,
// unknown account, amount beyond the balance) leave balances untouched; see
// the acceptance criteria in LESSON.md for which error each case carries.
func (b *Bank) Withdraw(account string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("withdraw %d from %q: amount must be positive", amount, account)
	}
	balance, ok := b.balances[account]
	if !ok {
		return fmt.Errorf("withdraw from %q: %w", account, ErrAccountNotFound)
	}
	if amount > balance {
		return fmt.Errorf("withdraw %d from %q: have %d: %w", amount, account, balance, ErrInsufficientFunds)
	}
	b.balances[account] = balance - amount
	return nil
}

// Transfer moves amount cents from one account to the other. On failure it
// adds "transfer" context with both account names while keeping the
// underlying cause detectable via errors.Is.
func (b *Bank) Transfer(from, to string, amount int) error {
	if err := b.Withdraw(from, amount); err != nil {
		return fmt.Errorf("transfer from %q to %q: %w", from, to, err)
	}
	if err := b.Deposit(to, amount); err != nil {
		return fmt.Errorf("transfer from %q to %q: %w", from, to, err)
	}
	return nil
}
