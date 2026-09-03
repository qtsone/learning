package tdd

import "testing"

// These tests are ordered as TDD increments. Treat each one as the "next
// failing test": make it green with the simplest change, refactor, move on.

func assertAdd(t *testing.T, input string, want int) {
	t.Helper()
	got, err := Add(input)
	if err != nil {
		t.Fatalf("Add(%q) returned unexpected error: %v", input, err)
	}
	if got != want {
		t.Errorf("Add(%q) = %d, want %d", input, got, want)
	}
}

func TestAddIncrement1EmptyStringIsZero(t *testing.T) {
	assertAdd(t, "", 0)
}

func TestAddIncrement2SingleNumber(t *testing.T) {
	assertAdd(t, "5", 5)
	assertAdd(t, "42", 42)
}

func TestAddIncrement3TwoNumbers(t *testing.T) {
	assertAdd(t, "1,2", 3)
	assertAdd(t, "10,20", 30)
}

func TestAddIncrement4AnyCount(t *testing.T) {
	assertAdd(t, "1,2,3,4", 10)
	assertAdd(t, "1,2,3,4,5,6,7,8,9,10", 55)
}

func TestAddIncrement5NewlinesAlsoSeparate(t *testing.T) {
	assertAdd(t, "1\n2,3", 6)
	assertAdd(t, "4\n5\n6", 15)
}

func TestAddIncrement6NegativesRejected(t *testing.T) {
	_, err := Add("1,-2,3,-4")
	if err == nil {
		t.Fatal(`Add("1,-2,3,-4") returned nil error, want an error naming every negative`)
	}
	want := "negative numbers not allowed: -2, -4"
	if got := err.Error(); got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}
