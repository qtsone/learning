package main

import "testing"

func TestZeroReport(t *testing.T) {
	want := `count=0 price=0 name="" active=false`
	if got := ZeroReport(); got != want {
		t.Errorf("ZeroReport() = %q, want %q (declare each variable with var and no value — Go supplies the zero value)", got, want)
	}
}

func TestAverage(t *testing.T) {
	cases := []struct {
		name  string
		sum   int
		count int
		want  float64
	}{
		{"seven halves", 7, 2, 3.5},
		{"nine quarters", 9, 4, 2.25},
		{"exact", 10, 5, 2},
		{"below one", 1, 2, 0.5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Average(c.sum, c.count); got != c.want {
				t.Errorf("Average(%d, %d) = %v, want %v (convert to float64 BEFORE dividing — integer division truncates)", c.sum, c.count, got, c.want)
			}
		})
	}
}

func TestPriceTag(t *testing.T) {
	cases := []struct {
		name  string
		item  string
		cents int
		want  string
	}{
		{"dollars and cents", "coffee", 350, "coffee: $3.50"},
		{"under a dollar", "apple", 99, "apple: $0.99"},
		{"single cents", "gum", 5, "gum: $0.05"},
		{"round amount", "lunch", 1000, "lunch: $10.00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := PriceTag(c.item, c.cents); got != c.want {
				t.Errorf("PriceTag(%q, %d) = %q, want %q (%%.2f always shows two decimals)", c.item, c.cents, got, c.want)
			}
		})
	}
}

func TestWeekdayConstants(t *testing.T) {
	days := []struct {
		name string
		got  int
		want int
	}{
		{"Monday", Monday, 1},
		{"Tuesday", Tuesday, 2},
		{"Wednesday", Wednesday, 3},
		{"Thursday", Thursday, 4},
		{"Friday", Friday, 5},
		{"Saturday", Saturday, 6},
		{"Sunday", Sunday, 7},
	}
	for _, d := range days {
		if d.got != d.want {
			t.Errorf("%s = %d, want %d (one iota expression on the first line; the following lines inherit it)", d.name, d.got, d.want)
		}
	}
}
