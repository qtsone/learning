package sliceops

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

// Celsius is a defined type on float64 — it satisfies Number only because
// the constraint uses ~float64, not bare float64.
type Celsius float64

type point struct{ X, Y int }

func (p point) String() string { return fmt.Sprintf("(%d,%d)", p.X, p.Y) }

func TestMap(t *testing.T) {
	t.Run("int to string", func(t *testing.T) {
		got := Map([]int{3, 1, 4}, strconv.Itoa)
		want := []string{"3", "1", "4"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Map([3 1 4], strconv.Itoa) = %v, want %v", got, want)
		}
	})
	t.Run("string to length", func(t *testing.T) {
		got := Map([]string{"go", "gopher"}, func(s string) int { return len(s) })
		want := []int{2, 6}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Map([go gopher], len) = %v, want %v", got, want)
		}
	})
	t.Run("empty input", func(t *testing.T) {
		got := Map([]int{}, strconv.Itoa)
		if len(got) != 0 {
			t.Errorf("Map of empty slice has len %d, want 0", len(got))
		}
	})
}

func TestMapExplicitInstantiation(t *testing.T) {
	// No call arguments here, so nothing to infer from: the type arguments
	// must be written out to use Map as a plain function value.
	double := Map[int, int]
	got := double([]int{1, 2, 3}, func(v int) int { return v * 2 })
	want := []int{2, 4, 6}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Map[int, int]([1 2 3], double) = %v, want %v", got, want)
	}
}

func TestFilter(t *testing.T) {
	t.Run("keep even ints", func(t *testing.T) {
		got := Filter([]int{1, 2, 3, 4, 5, 6}, func(v int) bool { return v%2 == 0 })
		want := []int{2, 4, 6}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Filter(evens) = %v, want %v", got, want)
		}
	})
	t.Run("keep long strings", func(t *testing.T) {
		got := Filter([]string{"go", "gopher", "generics"}, func(s string) bool { return len(s) > 3 })
		want := []string{"gopher", "generics"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Filter(len>3) = %v, want %v", got, want)
		}
	})
	t.Run("keep nothing", func(t *testing.T) {
		got := Filter([]int{1, 2, 3}, func(int) bool { return false })
		if len(got) != 0 {
			t.Errorf("Filter(keep nothing) = %v, want empty", got)
		}
	})
}

func TestIndexOf(t *testing.T) {
	ints := []struct {
		name   string
		s      []int
		target int
		want   int
	}{
		{"found in middle", []int{9, 3, 7, 3}, 7, 2},
		{"first occurrence wins", []int{9, 3, 7, 3}, 3, 1},
		{"found at start", []int{9, 3, 7}, 9, 0},
		{"not found", []int{9, 3, 7}, 5, -1},
		{"empty slice", nil, 1, -1},
	}
	for _, c := range ints {
		t.Run(c.name, func(t *testing.T) {
			if got := IndexOf(c.s, c.target); got != c.want {
				t.Errorf("IndexOf(%v, %d) = %d, want %d", c.s, c.target, got, c.want)
			}
		})
	}
	t.Run("works for strings too", func(t *testing.T) {
		if got := IndexOf([]string{"a", "b", "c"}, "c"); got != 2 {
			t.Errorf("IndexOf([a b c], c) = %d, want 2", got)
		}
	})
}

func TestSum(t *testing.T) {
	t.Run("ints", func(t *testing.T) {
		if got := Sum([]int{1, 2, 3, 4}); got != 10 {
			t.Errorf("Sum([1 2 3 4]) = %d, want 10", got)
		}
	})
	t.Run("float64s", func(t *testing.T) {
		if got := Sum([]float64{0.5, 1.25, 2.25}); got != 4.0 {
			t.Errorf("Sum([0.5 1.25 2.25]) = %v, want 4", got)
		}
	})
	t.Run("defined type via tilde", func(t *testing.T) {
		temps := []Celsius{20.5, 1.5, -2}
		var want Celsius = 20
		if got := Sum(temps); got != want {
			t.Errorf("Sum(%v) = %v, want %v", temps, got, want)
		}
	})
	t.Run("empty is zero", func(t *testing.T) {
		if got := Sum([]int{}); got != 0 {
			t.Errorf("Sum(empty) = %d, want 0", got)
		}
	})
}

func TestDescribeAll(t *testing.T) {
	t.Run("points", func(t *testing.T) {
		got := DescribeAll([]point{{1, 2}, {3, 4}})
		want := []string{"(1,2)", "(3,4)"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("DescribeAll(points) = %v, want %v", got, want)
		}
	})
	t.Run("empty input", func(t *testing.T) {
		got := DescribeAll([]point{})
		if len(got) != 0 {
			t.Errorf("DescribeAll(empty) = %v, want empty", got)
		}
	})
}
