package stats

import "testing"

func TestLongest(t *testing.T) {
	words := []string{"go", "gopher", "tea"}
	if got := Longest(words); got != "gopher" {
		t.Errorf("Longest(%v) = %q, want %q", words, got, "gopher")
	}
	tie := []string{"one", "two", "six"}
	if got := Longest(tie); got != "one" {
		t.Errorf("Longest(%v) = %q, want %q (earliest longest wins)", tie, got, "one")
	}
	if got := Longest([]string{}); got != "" {
		t.Errorf("Longest([]) = %q, want %q", got, "")
	}
}

func TestCountVowels(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"simple word", "gopher", 2},
		{"no vowels", "rhythm", 0},
		{"empty string", "", 0},
		{"mixed case", "AEIOU and aeiou", 11},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CountVowels(c.in); got != c.want {
				t.Errorf("CountVowels(%q) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestAverage(t *testing.T) {
	cases := []struct {
		name    string
		in      []int
		want    float64
		wantErr bool
	}{
		{"divides evenly", []int{2, 4, 6}, 4, false},
		{"does not divide evenly", []int{1, 2}, 1.5, false},
		{"single value", []int{7}, 7, false},
		{"empty slice", []int{}, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Average(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("Average(%v) = %v, want an error", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Average(%v) returned unexpected error: %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("Average(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
