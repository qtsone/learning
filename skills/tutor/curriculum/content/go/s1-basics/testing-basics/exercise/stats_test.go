package stats

import "testing"

// Replace each t.Fatal stub below with a real test. The acceptance criteria
// in LESSON.md are the contract; the comments here say what each test must
// cover at minimum.

func TestLongest(t *testing.T) {
	// TODO: write this one plainly — no table yet. Call Longest, compare
	// got to want with if, report mismatches with t.Errorf, including the
	// input, got, and want in the message.
	// Cover at least: a normal slice, a tie (earliest longest word wins),
	// and an empty slice.
	t.Fatal("TODO: write TestLongest")
}

func TestCountVowels(t *testing.T) {
	// TODO: make this table-driven — a slice of anonymous structs
	// (name, in, want) and one named t.Run subtest per case.
	// Cover at least: a simple word, a word with no vowels, the empty
	// string, and mixed upper/lower case.
	t.Fatal("TODO: write the table-driven TestCountVowels")
}

func TestAverage(t *testing.T) {
	// TODO: table-driven again, now with error checking.
	// Cover at least: a slice that divides evenly, a slice whose true
	// average is NOT a whole number, and the empty slice, which must
	// return an error. Choose cases where a wrong answer can't hide.
	t.Fatal("TODO: write the table-driven TestAverage")
}
