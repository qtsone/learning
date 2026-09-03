package bank

import (
	"errors"
	"strings"
	"testing"
)

func TestDepositAndBalance(t *testing.T) {
	b := NewBank()
	if err := b.Deposit("alice", 100); err != nil {
		t.Fatalf("Deposit(\"alice\", 100) = %v, want nil", err)
	}
	if err := b.Deposit("alice", 50); err != nil {
		t.Fatalf("second Deposit(\"alice\", 50) = %v, want nil", err)
	}
	got, err := b.Balance("alice")
	if err != nil {
		t.Fatalf("Balance(\"alice\") after deposits = error %v, want nil", err)
	}
	if got != 150 {
		t.Errorf("Balance(\"alice\") = %d, want 150 (100 + 50)", got)
	}
}

func TestDepositRejectsNonPositive(t *testing.T) {
	b := NewBank()
	for _, amount := range []int{0, -50} {
		if err := b.Deposit("alice", amount); err == nil {
			t.Errorf("Deposit(\"alice\", %d) = nil, want an error: non-positive amounts must be rejected", amount)
		}
	}
	if _, err := b.Balance("alice"); !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("after only rejected deposits, Balance(\"alice\") error = %v, want ErrAccountNotFound: a failed deposit must not create the account", err)
	}
}

func TestBalanceUnknownAccount(t *testing.T) {
	b := NewBank()
	_, err := b.Balance("ghost")
	if err == nil {
		t.Fatalf("Balance(\"ghost\") on an empty bank = nil error, want an error")
	}
	if !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("Balance(\"ghost\") error = %v; errors.Is(err, ErrAccountNotFound) = false, want true — wrap the sentinel with %%w", err)
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("Balance(\"ghost\") error = %q, want the account name \"ghost\" in the message", err)
	}
}

func TestWithdraw(t *testing.T) {
	b := NewBank()
	if err := b.Deposit("alice", 200); err != nil {
		t.Fatalf("setup: Deposit(\"alice\", 200) = %v, want nil", err)
	}
	if err := b.Withdraw("alice", 80); err != nil {
		t.Fatalf("Withdraw(\"alice\", 80) with balance 200 = %v, want nil", err)
	}
	got, err := b.Balance("alice")
	if err != nil {
		t.Fatalf("Balance(\"alice\") after withdraw = error %v, want nil", err)
	}
	if got != 120 {
		t.Errorf("after withdrawing 80 from 200, Balance(\"alice\") = %d, want 120", got)
	}
}

func TestWithdrawErrors(t *testing.T) {
	cases := []struct {
		name     string
		account  string
		amount   int
		sentinel error // nil: any error is fine, no sentinel required
	}{
		{"unknown account", "ghost", 10, ErrAccountNotFound},
		{"insufficient funds", "alice", 500, ErrInsufficientFunds},
		{"zero amount", "alice", 0, nil},
		{"negative amount", "alice", -10, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := NewBank()
			if err := b.Deposit("alice", 100); err != nil {
				t.Fatalf("setup: Deposit(\"alice\", 100) = %v, want nil", err)
			}
			err := b.Withdraw(c.account, c.amount)
			if err == nil {
				t.Fatalf("Withdraw(%q, %d) = nil, want an error", c.account, c.amount)
			}
			if c.sentinel != nil && !errors.Is(err, c.sentinel) {
				t.Errorf("Withdraw(%q, %d) error = %v; errors.Is with sentinel %v = false, want true — wrap it with %%w", c.account, c.amount, err, c.sentinel)
			}
			if got, _ := b.Balance("alice"); got != 100 {
				t.Errorf("after a failed withdraw, Balance(\"alice\") = %d, want 100 unchanged", got)
			}
		})
	}
}

func TestWithdrawMessageHasContext(t *testing.T) {
	b := NewBank()
	if err := b.Deposit("alice", 100); err != nil {
		t.Fatalf("setup: Deposit(\"alice\", 100) = %v, want nil", err)
	}
	err := b.Withdraw("alice", 500)
	if err == nil {
		t.Fatalf("Withdraw(\"alice\", 500) with balance 100 = nil, want an error")
	}
	for _, want := range []string{"alice", "500", "100"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Withdraw error = %q, want the message to mention %q (account, amount, and current balance)", err, want)
		}
	}
}

func TestTransfer(t *testing.T) {
	b := NewBank()
	if err := b.Deposit("alice", 300); err != nil {
		t.Fatalf("setup: Deposit(\"alice\", 300) = %v, want nil", err)
	}
	if err := b.Transfer("alice", "bob", 120); err != nil {
		t.Fatalf("Transfer(\"alice\", \"bob\", 120) with balance 300 = %v, want nil", err)
	}
	if got, _ := b.Balance("alice"); got != 180 {
		t.Errorf("after transfer, Balance(\"alice\") = %d, want 180", got)
	}
	if got, _ := b.Balance("bob"); got != 120 {
		t.Errorf("after transfer, Balance(\"bob\") = %d, want 120", got)
	}
}

func TestTransferKeepsCauseInspectable(t *testing.T) {
	cases := []struct {
		name     string
		from     string
		amount   int
		sentinel error
	}{
		{"insufficient funds", "alice", 100, ErrInsufficientFunds},
		{"unknown sender", "ghost", 10, ErrAccountNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := NewBank()
			if err := b.Deposit("alice", 50); err != nil {
				t.Fatalf("setup: Deposit(\"alice\", 50) = %v, want nil", err)
			}
			err := b.Transfer(c.from, "bob", c.amount)
			if err == nil {
				t.Fatalf("Transfer(%q, \"bob\", %d) = nil, want an error", c.from, c.amount)
			}
			if !errors.Is(err, c.sentinel) {
				t.Errorf("Transfer error = %v; errors.Is with sentinel %v = false, want true — every layer must wrap with %%w", err, c.sentinel)
			}
			for _, want := range []string{"transfer", c.from, "bob"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Transfer error = %q, want its own context in the message: the word \"transfer\" and both account names (missing %q)", err, want)
				}
			}
			if got, _ := b.Balance("alice"); got != 50 {
				t.Errorf("after a failed transfer, Balance(\"alice\") = %d, want 50 unchanged", got)
			}
			if _, err := b.Balance("bob"); !errors.Is(err, ErrAccountNotFound) {
				t.Errorf("after a failed transfer, Balance(\"bob\") error = %v, want ErrAccountNotFound: a failed transfer must not create the recipient", err)
			}
		})
	}
}
