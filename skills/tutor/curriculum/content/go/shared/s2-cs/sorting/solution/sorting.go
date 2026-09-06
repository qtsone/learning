package sorting

import (
	"cmp"
	"slices"
)

// InsertionSort sorts nums ascending, in place.
// It must be adaptive: linear time on already-sorted input.
func InsertionSort(nums []int) {
	for i := 1; i < len(nums); i++ {
		key := nums[i]
		j := i - 1
		for j >= 0 && nums[j] > key {
			nums[j+1] = nums[j]
			j--
		}
		nums[j+1] = key
	}
}

// Merge combines two individually sorted slices into one sorted slice,
// taking from a on ties (that choice keeps merge sort stable).
func Merge(a, b []int) []int {
	out := make([]int, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	out = append(out, a[i:]...)
	out = append(out, b[j:]...)
	return out
}

// MergeSort returns a new ascending-sorted slice and leaves nums untouched.
func MergeSort(nums []int) []int {
	if len(nums) <= 1 {
		return slices.Clone(nums)
	}
	mid := len(nums) / 2
	return Merge(MergeSort(nums[:mid]), MergeSort(nums[mid:]))
}

// Player is a scoreboard entry.
type Player struct {
	Name  string
	Score int
}

// RankPlayers sorts players in place: Score descending, ties by Name
// ascending. Use the standard library with a custom comparison.
func RankPlayers(players []Player) {
	slices.SortFunc(players, func(a, b Player) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
}

// SortByScore sorts players in place by Score descending only.
// Players with equal scores must keep their original relative order.
func SortByScore(players []Player) {
	slices.SortStableFunc(players, func(a, b Player) int {
		return cmp.Compare(b.Score, a.Score)
	})
}
