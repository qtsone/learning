package list

import (
	"slices"
	"testing"
	"time"
)

func TestNewIsEmpty(t *testing.T) {
	l := New()
	if got := l.Len(); got != 0 {
		t.Errorf("New().Len() = %d, want 0", got)
	}
	if got := l.Values(); len(got) != 0 {
		t.Errorf("New().Values() = %v, want empty", got)
	}
	if l.Contains(42) {
		t.Error("New().Contains(42) = true, want false — nothing was added")
	}
	if l.Delete(42) {
		t.Error("New().Delete(42) = true, want false — nothing to remove")
	}
}

func TestAppendKeepsOrder(t *testing.T) {
	l := New()
	for _, v := range []int{10, 20, 30} {
		l.Append(v)
	}
	if got, want := l.Values(), []int{10, 20, 30}; !slices.Equal(got, want) {
		t.Errorf("Values() after appends = %v, want %v (Append adds to the back)", got, want)
	}
	if got := l.Len(); got != 3 {
		t.Errorf("Len() = %d, want 3", got)
	}
}

func TestPrependReversesOrder(t *testing.T) {
	l := New()
	for _, v := range []int{10, 20, 30} {
		l.Prepend(v)
	}
	if got, want := l.Values(), []int{30, 20, 10}; !slices.Equal(got, want) {
		t.Errorf("Values() after prepends = %v, want %v (Prepend adds to the front)", got, want)
	}
}

func TestMixedAppendPrepend(t *testing.T) {
	l := New()
	l.Append(2)
	l.Prepend(1)
	l.Append(3)
	l.Prepend(0)
	if got, want := l.Values(), []int{0, 1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("Values() = %v, want %v", got, want)
	}
	if got := l.Len(); got != 4 {
		t.Errorf("Len() = %d, want 4", got)
	}
}

func TestPrependOnEmptyThenAppend(t *testing.T) {
	l := New()
	l.Prepend(1)
	l.Append(2)
	if got, want := l.Values(), []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("Values() = %v, want %v — did Prepend on an empty list set the tail?", got, want)
	}
}

func TestContains(t *testing.T) {
	l := New()
	for _, v := range []int{10, 20, 30} {
		l.Append(v)
	}
	cases := []struct {
		name string
		v    int
		want bool
	}{
		{"head value", 10, true},
		{"middle value", 20, true},
		{"tail value", 30, true},
		{"absent value", 99, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := l.Contains(c.v); got != c.want {
				t.Errorf("Contains(%d) = %t, want %t", c.v, got, c.want)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	cases := []struct {
		name       string
		start      []int
		deleteV    int
		wantOK     bool
		wantValues []int
	}{
		{"head", []int{1, 2, 3}, 1, true, []int{2, 3}},
		{"middle", []int{1, 2, 3}, 2, true, []int{1, 3}},
		{"tail", []int{1, 2, 3}, 3, true, []int{1, 2}},
		{"only element", []int{7}, 7, true, []int{}},
		{"absent value", []int{1, 2, 3}, 9, false, []int{1, 2, 3}},
		{"first occurrence only", []int{5, 1, 5, 2}, 5, true, []int{1, 5, 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := New()
			for _, v := range c.start {
				l.Append(v)
			}
			if got := l.Delete(c.deleteV); got != c.wantOK {
				t.Fatalf("Delete(%d) = %t, want %t", c.deleteV, got, c.wantOK)
			}
			if got := l.Values(); !slices.Equal(got, c.wantValues) {
				t.Errorf("Values() after Delete(%d) = %v, want %v", c.deleteV, got, c.wantValues)
			}
			if got := l.Len(); got != len(c.wantValues) {
				t.Errorf("Len() after Delete(%d) = %d, want %d", c.deleteV, got, len(c.wantValues))
			}
		})
	}
}

func TestAppendAfterDeletingTail(t *testing.T) {
	l := New()
	l.Append(1)
	l.Append(2)
	l.Append(3)
	l.Delete(3)
	l.Append(4)
	if got, want := l.Values(), []int{1, 2, 4}; !slices.Equal(got, want) {
		t.Errorf("Values() = %v, want %v — Delete of the last node must move the tail pointer back, or Append attaches to a removed node", got, want)
	}
}

func TestAppendAfterDeletingOnlyElement(t *testing.T) {
	l := New()
	l.Append(1)
	l.Delete(1)
	l.Append(2)
	if got, want := l.Values(), []int{2}; !slices.Equal(got, want) {
		t.Errorf("Values() = %v, want %v — deleting the only element must clear both head and tail", got, want)
	}
}

func TestManyAppendsStayFast(t *testing.T) {
	const n = 150_000
	l := New()
	start := time.Now()
	for i := range n {
		l.Append(i)
	}
	elapsed := time.Since(start)
	if got := l.Len(); got != n {
		t.Fatalf("Len() after %d appends = %d, want %d", n, got, n)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("%d appends took %v — Append looks O(n) per call; use the tail pointer instead of walking to the end", n, elapsed)
	}
}
