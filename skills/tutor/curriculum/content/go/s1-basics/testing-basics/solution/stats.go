// Package stats holds small text and number helpers.
//
// The functions below are already written — this lesson you write the tests.
// The doc comments are the contract; one implementation doesn't live up to
// its comment. Your tests will tell you which.
package stats

import (
	"errors"
	"strings"
)

// Longest returns the longest word in words.
// On a tie the earliest longest word wins; an empty slice returns "".
func Longest(words []string) string {
	longest := ""
	for _, w := range words {
		if len(w) > len(longest) {
			longest = w
		}
	}
	return longest
}

// CountVowels returns how many vowels appear in s.
// Vowels are a, e, i, o, u — upper or lower case.
func CountVowels(s string) int {
	count := 0
	for _, r := range s {
		if strings.ContainsRune("aeiouAEIOU", r) {
			count++
		}
	}
	return count
}

// Average returns the arithmetic mean of nums.
// The empty slice has no average, so it returns an error.
func Average(nums []int) (float64, error) {
	if len(nums) == 0 {
		return 0, errors.New("average of empty slice")
	}
	sum := 0
	for _, n := range nums {
		sum += n
	}
	// The starter's bug: float64(sum / len(nums)) divides ints first,
	// truncating the remainder. Convert before dividing.
	return float64(sum) / float64(len(nums)), nil
}
