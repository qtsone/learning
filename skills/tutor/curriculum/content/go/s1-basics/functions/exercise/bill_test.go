package main

import "testing"

func TestSum(t *testing.T) {
	cases := []struct {
		name   string
		prices []int
		want   int
	}{
		{"no prices", nil, 0},
		{"one price", []int{999}, 999},
		{"several prices", []int{1250, 899, 1451}, 3600},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Sum(c.prices...); got != c.want {
				t.Errorf("Sum(%v) = %d, want %d", c.prices, got, c.want)
			}
		})
	}
}

func TestSumDirectArguments(t *testing.T) {
	if got := Sum(100, 200, 300); got != 600 {
		t.Errorf("Sum(100, 200, 300) = %d, want 600", got)
	}
}

func TestSplit(t *testing.T) {
	cases := []struct {
		name          string
		total, people int
		wantShare     int
		wantRemainder int
		wantErr       bool
	}{
		{"even split", 3000, 3, 1000, 0, false},
		{"remainder left over", 1000, 3, 333, 1, false},
		{"one person takes all", 999, 1, 999, 0, false},
		{"zero people", 500, 0, 0, 0, true},
		{"negative people", 500, -2, 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			share, remainder, err := Split(c.total, c.people)
			if c.wantErr {
				if err == nil {
					t.Fatalf("Split(%d, %d) returned a nil error, want an error for the bad input", c.total, c.people)
				}
				return
			}
			if err != nil {
				t.Fatalf("Split(%d, %d) returned an unexpected error: %v", c.total, c.people, err)
			}
			if share != c.wantShare || remainder != c.wantRemainder {
				t.Errorf("Split(%d, %d) = (%d, %d), want (%d, %d)",
					c.total, c.people, share, remainder, c.wantShare, c.wantRemainder)
			}
		})
	}
}

func TestSplitBill(t *testing.T) {
	cases := []struct {
		name          string
		people        int
		prices        []int
		wantShare     int
		wantRemainder int
		wantErr       bool
	}{
		{"dinner for three", 3, []int{1250, 899, 1451}, 1200, 0, false},
		{"coffee for two with remainder", 2, []int{375, 420}, 397, 1, false},
		{"no prices still splits to zero", 4, nil, 0, 0, false},
		{"zero people", 0, []int{500}, 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			share, remainder, err := SplitBill(c.people, c.prices...)
			if c.wantErr {
				if err == nil {
					t.Fatalf("SplitBill(%d, %v) returned a nil error, want an error for the bad input", c.people, c.prices)
				}
				return
			}
			if err != nil {
				t.Fatalf("SplitBill(%d, %v) returned an unexpected error: %v", c.people, c.prices, err)
			}
			if share != c.wantShare || remainder != c.wantRemainder {
				t.Errorf("SplitBill(%d, %v) = (%d, %d), want (%d, %d)",
					c.people, c.prices, share, remainder, c.wantShare, c.wantRemainder)
			}
		})
	}
}

func TestFormatCents(t *testing.T) {
	cases := []struct {
		name  string
		cents int
		want  string
	}{
		{"dollars and cents", 1250, "12.50"},
		{"cents only", 5, "0.05"},
		{"round dollars", 300, "3.00"},
		{"zero", 0, "0.00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FormatCents(c.cents); got != c.want {
				t.Errorf("FormatCents(%d) = %q, want %q", c.cents, got, c.want)
			}
		})
	}
}
