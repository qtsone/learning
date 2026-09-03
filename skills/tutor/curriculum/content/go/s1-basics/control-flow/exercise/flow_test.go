package main

import "testing"

func TestSign(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{-5, "negative"},
		{-1, "negative"},
		{0, "zero"},
		{1, "positive"},
		{42, "positive"},
	}
	for _, c := range cases {
		if got := Sign(c.in); got != c.want {
			t.Errorf("Sign(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAward(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{1, "gold"},
		{2, "silver"},
		{3, "bronze"},
		{0, "none"},
		{4, "none"},
		{-1, "none"},
	}
	for _, c := range cases {
		if got := Award(c.in); got != c.want {
			t.Errorf("Award(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSumEvens(t *testing.T) {
	cases := []struct {
		limit, want int
	}{
		{0, 0},
		{1, 0},
		{2, 2},
		{7, 12},
		{10, 30},
		{100, 2550},
	}
	for _, c := range cases {
		if got := SumEvens(c.limit); got != c.want {
			t.Errorf("SumEvens(%d) = %d, want %d (sum of the even numbers 1..%d)",
				c.limit, got, c.want, c.limit)
		}
	}
}

func TestRepeat(t *testing.T) {
	cases := []struct {
		word  string
		times int
		want  string
	}{
		{"go", 3, "gogogo"},
		{"x", 1, "x"},
		{"na", 0, ""},
		{"y", -2, ""},
	}
	for _, c := range cases {
		if got := Repeat(c.word, c.times); got != c.want {
			t.Errorf("Repeat(%q, %d) = %q, want %q", c.word, c.times, got, c.want)
		}
	}
}

func TestCollatzSteps(t *testing.T) {
	cases := []struct {
		n, want int
	}{
		{1, 0},
		{2, 1},
		{6, 8},
		{7, 16},
		{27, 111},
	}
	for _, c := range cases {
		if got := CollatzSteps(c.n); got != c.want {
			t.Errorf("CollatzSteps(%d) = %d, want %d steps to reach 1", c.n, got, c.want)
		}
	}
}

func TestFirstPowerAbove(t *testing.T) {
	cases := []struct {
		limit, want int
	}{
		{0, 1},
		{1, 2},
		{5, 8},
		{16, 32},
		{100, 128},
	}
	for _, c := range cases {
		if got := FirstPowerAbove(c.limit); got != c.want {
			t.Errorf("FirstPowerAbove(%d) = %d, want %d (smallest power of two > %d)",
				c.limit, got, c.want, c.limit)
		}
	}
}

func TestCountPrimes(t *testing.T) {
	cases := []struct {
		limit, want int
	}{
		{1, 0},
		{2, 1},
		{10, 4},
		{20, 8},
		{100, 25},
	}
	for _, c := range cases {
		if got := CountPrimes(c.limit); got != c.want {
			t.Errorf("CountPrimes(%d) = %d, want %d primes in 2..%d",
				c.limit, got, c.want, c.limit)
		}
	}
}
