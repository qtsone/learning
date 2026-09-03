package bigo

// Part A — analyze. The eight functions below are already written and work.
// Your job is to READ them, count how the work grows with the input, and
// record each one's time complexity in the Complexities map at the bottom,
// using these constants:

const (
	O1     = "O(1)"
	OLogN  = "O(log n)"
	ON     = "O(n)"
	ONLogN = "O(n log n)"
	ON2    = "O(n^2)"
	O2N    = "O(2^n)"
)

// First returns the first element, or 0 for an empty slice.
func First(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	return xs[0]
}

// Sum adds up every element.
func Sum(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}

// SumTwice walks the slice twice: once to sum, once to count.
func SumTwice(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	count := 0
	for range xs {
		count++
	}
	return total + count
}

// HasPairSum reports whether any two distinct elements add up to target.
func HasPairSum(xs []int, target int) bool {
	for i := 0; i < len(xs); i++ {
		for j := i + 1; j < len(xs); j++ {
			if xs[i]+xs[j] == target {
				return true
			}
		}
	}
	return false
}

// Halving counts how many times n can be halved before reaching 1.
func Halving(n int) int {
	steps := 0
	for n > 1 {
		n /= 2
		steps++
	}
	return steps
}

// HalvingPerItem runs a halving countdown once per element.
func HalvingPerItem(xs []int) int {
	steps := 0
	for range xs {
		for m := len(xs); m > 1; m /= 2 {
			steps++
		}
	}
	return steps
}

// FirstTen sums, for each element, up to the first ten elements of xs.
func FirstTen(xs []int) int {
	total := 0
	for range xs {
		for j := 0; j < 10 && j < len(xs); j++ {
			total += xs[j]
		}
	}
	return total
}

// Combos counts every yes/no choice pattern over n items: total starts at 1
// and doubles once per item, then the loop visits each pattern.
func Combos(n int) int {
	total := 1
	for i := 0; i < n; i++ {
		total *= 2
	}
	count := 0
	for i := 0; i < total; i++ {
		count++
	}
	return count
}

// Complexities maps each function above to its time complexity.
// Sequential passes stay linear (SumTwice), a constant-bounded inner loop
// stays linear (FirstTen) — only loops that both scale with n multiply.
var Complexities = map[string]string{
	"First":          O1,
	"Sum":            ON,
	"SumTwice":       ON,
	"HasPairSum":     ON2,
	"Halving":        OLogN,
	"HalvingPerItem": ONLogN,
	"FirstTen":       ON,
	"Combos":         O2N,
}
