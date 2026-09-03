// Package patterns is the CS-fundamentals capstone: two pointers, sliding
// windows, and a mixed problem set drawing on the whole stage.
package patterns

// PairSum reports two indexes i < j with nums[i]+nums[j] == target.
// nums is sorted ascending. When no such pair exists it returns (0, 0, false).
// Required: O(n) time, O(1) extra space — no map.
func PairSum(nums []int, target int) (int, int, bool) {
	lo, hi := 0, len(nums)-1
	for lo < hi {
		switch sum := nums[lo] + nums[hi]; {
		case sum == target:
			return lo, hi, true
		case sum < target:
			// nums[hi] is the largest partner left; if even it falls
			// short, nums[lo] can never be in a pair.
			lo++
		default:
			// nums[lo] is the smallest partner left; nums[hi] overshoots
			// with everyone.
			hi--
		}
	}
	return 0, 0, false
}

// RemoveDuplicates compacts a sorted slice in place so that its first k
// elements are the distinct values in order, and returns k.
// Required: O(n) time, no allocations.
func RemoveDuplicates(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	// Invariant: nums[0..w) is the deduplicated output so far, and the
	// reader r never lags behind the writer w.
	w := 1
	for r := 1; r < len(nums); r++ {
		if nums[r] != nums[w-1] {
			nums[w] = nums[r]
			w++
		}
	}
	return w
}

// MaxWindowSum returns the largest sum of any k consecutive elements and
// true. It returns (0, false) when k <= 0 or k > len(nums).
// Required: O(n) time regardless of k.
func MaxWindowSum(nums []int, k int) (int, bool) {
	if k <= 0 || k > len(nums) {
		return 0, false
	}
	sum := 0
	for _, v := range nums[:k] {
		sum += v
	}
	best := sum
	for i := k; i < len(nums); i++ {
		sum += nums[i] - nums[i-k]
		best = max(best, sum)
	}
	return best, true
}

// LongestUniqueRun returns the length in runes of the longest substring of s
// containing no repeated rune.
// Required: O(n) time.
func LongestUniqueRun(s string) int {
	runes := []rune(s) // runes, not bytes: multibyte characters count once
	last := make(map[rune]int, len(runes))
	best, left := 0, 0
	for right, r := range runes {
		// The guard prev >= left ignores sightings from before the current
		// window — jumping back to one would silently widen it.
		if prev, seen := last[r]; seen && prev >= left {
			left = prev + 1
		}
		last[r] = right
		best = max(best, right-left+1)
	}
	return best
}
