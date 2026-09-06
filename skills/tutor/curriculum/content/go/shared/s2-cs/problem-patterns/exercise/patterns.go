// Package patterns is the CS-fundamentals capstone: two pointers, sliding
// windows, and a mixed problem set drawing on the whole stage.
package patterns

// PairSum reports two indexes i < j with nums[i]+nums[j] == target.
// nums is sorted ascending. When no such pair exists it returns (0, 0, false).
// Required: O(n) time, O(1) extra space — no map.
func PairSum(nums []int, target int) (int, int, bool) {
	// TODO: converge two pointers from the ends (LESSON.md, flavor one).
	return 0, 0, false
}

// RemoveDuplicates compacts a sorted slice in place so that its first k
// elements are the distinct values in order, and returns k.
// Required: O(n) time, no allocations.
func RemoveDuplicates(nums []int) int {
	// TODO: reader and writer indexes (LESSON.md, flavor two).
	return 0
}

// MaxWindowSum returns the largest sum of any k consecutive elements and
// true. It returns (0, false) when k <= 0 or k > len(nums).
// Required: O(n) time regardless of k.
func MaxWindowSum(nums []int, k int) (int, bool) {
	// TODO: compute the first window once, then slide it.
	return 0, false
}

// LongestUniqueRun returns the length in runes of the longest substring of s
// containing no repeated rune.
// Required: O(n) time.
func LongestUniqueRun(s string) int {
	// TODO: variable window over []rune(s), with a map of last sightings.
	return 0
}
