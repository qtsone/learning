package pantry

import (
	"maps"
	"slices"
	"testing"
)

func TestCount(t *testing.T) {
	cases := []struct {
		name  string
		words []string
		want  map[string]int
	}{
		{"no words", nil, map[string]int{}},
		{"single word", []string{"go"}, map[string]int{"go": 1}},
		{"repeats counted", []string{"go", "maps", "go", "go"}, map[string]int{"go": 3, "maps": 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Count(c.words); !maps.Equal(got, c.want) {
				t.Errorf("Count(%v) = %v, want %v", c.words, got, c.want)
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	pantry := map[string]int{"flour": 2, "eggs": 0}
	cases := []struct {
		name string
		item string
		want string
	}{
		{"stocked", "flour", "2 in stock"},
		{"tracked but none left", "eggs", "out of stock"},
		{"never stocked", "milk", "not stocked"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Describe(pantry, c.item); got != c.want {
				t.Errorf("Describe(pantry, %q) = %q, want %q", c.item, got, c.want)
			}
		})
	}
}

func TestTake(t *testing.T) {
	t.Run("takes from stock", func(t *testing.T) {
		pantry := map[string]int{"flour": 3}
		if !Take(pantry, "flour", 2) {
			t.Fatal(`Take(pantry, "flour", 2) = false, want true`)
		}
		if got := pantry["flour"]; got != 1 {
			t.Errorf(`after Take, pantry["flour"] = %d, want 1`, got)
		}
	})
	t.Run("taking the last unit deletes the entry", func(t *testing.T) {
		pantry := map[string]int{"eggs": 2}
		if !Take(pantry, "eggs", 2) {
			t.Fatal(`Take(pantry, "eggs", 2) = false, want true`)
		}
		if _, ok := pantry["eggs"]; ok {
			t.Errorf(`after taking all eggs, pantry still has an "eggs" entry: %v`, pantry)
		}
	})
	t.Run("insufficient stock leaves pantry untouched", func(t *testing.T) {
		pantry := map[string]int{"eggs": 1}
		if Take(pantry, "eggs", 3) {
			t.Fatal(`Take(pantry, "eggs", 3) = true, want false`)
		}
		if got := pantry["eggs"]; got != 1 {
			t.Errorf(`after failed Take, pantry["eggs"] = %d, want 1 (unchanged)`, got)
		}
	})
	t.Run("missing item", func(t *testing.T) {
		pantry := map[string]int{"flour": 3}
		if Take(pantry, "milk", 1) {
			t.Fatal(`Take(pantry, "milk", 1) = true, want false`)
		}
		if len(pantry) != 1 {
			t.Errorf("after failed Take, pantry = %v, want it unchanged", pantry)
		}
	})
}

func TestSortedItems(t *testing.T) {
	cases := []struct {
		name   string
		pantry map[string]int
		want   []string
	}{
		{"empty pantry", map[string]int{}, nil},
		{"alphabetical order", map[string]int{"milk": 1, "eggs": 12, "flour": 2}, []string{"eggs", "flour", "milk"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SortedItems(c.pantry); !slices.Equal(got, c.want) {
				t.Errorf("SortedItems(%v) = %v, want %v", c.pantry, got, c.want)
			}
		})
	}
}

func TestNewSetAndHas(t *testing.T) {
	set := NewSet([]string{"vegan", "gluten-free", "vegan"})
	if len(set) != 2 {
		t.Fatalf("NewSet with a duplicate has %d entries, want 2 (duplicates collapse)", len(set))
	}
	for _, item := range []string{"vegan", "gluten-free"} {
		if !Has(set, item) {
			t.Errorf("Has(set, %q) = false, want true", item)
		}
	}
	if Has(set, "dairy") {
		t.Errorf(`Has(set, "dairy") = true, want false`)
	}
}
