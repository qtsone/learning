package bank

import "testing"

func TestCanAfford(t *testing.T) {
	a := Account{Owner: "Ada", Balance: 100}
	cases := []struct {
		name   string
		amount int
		want   bool
	}{
		{"less than the balance", 30, true},
		{"exactly the balance", 100, true},
		{"more than the balance", 101, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := a.CanAfford(c.amount); got != c.want {
				t.Errorf("Account{Balance: 100}.CanAfford(%d) = %v, want %v", c.amount, got, c.want)
			}
		})
	}
}

func TestDepositReachesTheCaller(t *testing.T) {
	a := Account{Owner: "Ada", Balance: 100}
	a.Deposit(50)
	if a.Balance != 150 {
		t.Errorf("after Deposit(50) on balance 100: Balance = %d, want 150 (did the method change a copy?)", a.Balance)
	}
}

func TestDepositIgnoresNegativeAmounts(t *testing.T) {
	a := Account{Owner: "Ada", Balance: 100}
	a.Deposit(-40)
	if a.Balance != 100 {
		t.Errorf("after Deposit(-40) on balance 100: Balance = %d, want 100 (negative deposits are ignored)", a.Balance)
	}
}

func TestWithdraw(t *testing.T) {
	cases := []struct {
		name        string
		start       int
		amount      int
		wantOK      bool
		wantBalance int
	}{
		{"affordable", 100, 30, true, 70},
		{"exactly the balance", 100, 100, true, 0},
		{"more than the balance", 100, 101, false, 100},
		{"negative amount", 100, -40, false, 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := Account{Owner: "Ada", Balance: c.start}
			if ok := a.Withdraw(c.amount); ok != c.wantOK {
				t.Errorf("Withdraw(%d) on balance %d = %v, want %v", c.amount, c.start, ok, c.wantOK)
			}
			if a.Balance != c.wantBalance {
				t.Errorf("after Withdraw(%d) on balance %d: Balance = %d, want %d", c.amount, c.start, a.Balance, c.wantBalance)
			}
		})
	}
}

func TestDollars(t *testing.T) {
	cases := []struct {
		name  string
		cents Cents
		want  string
	}{
		{"dollars and cents", 1234, "$12.34"},
		{"cents only", 5, "$0.05"},
		{"round dollars", 300, "$3.00"},
		{"zero", 0, "$0.00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.cents.Dollars(); got != c.want {
				t.Errorf("Cents(%d).Dollars() = %q, want %q", int(c.cents), got, c.want)
			}
		})
	}
}
