package sorting

// InsertionSort sorts nums ascending, in place.
// It must be adaptive: linear time on already-sorted input.
func InsertionSort(nums []int) {
	// TODO: implement per the acceptance criteria in LESSON.md.
}

// Merge combines two individually sorted slices into one sorted slice,
// taking from a on ties (that choice keeps merge sort stable).
func Merge(a, b []int) []int {
	// TODO: walk both slices front-to-front, always taking the smaller head.
	return nil
}

// MergeSort returns a new ascending-sorted slice and leaves nums untouched.
func MergeSort(nums []int) []int {
	// TODO: base case, split, recurse, Merge.
	return nil
}

// Player is a scoreboard entry.
type Player struct {
	Name  string
	Score int
}

// RankPlayers sorts players in place: Score descending, ties by Name
// ascending. Use the standard library with a custom comparison.
func RankPlayers(players []Player) {
	// TODO: slices.SortFunc with a multi-key comparison (see LESSON.md).
}

// SortByScore sorts players in place by Score descending only.
// Players with equal scores must keep their original relative order.
func SortByScore(players []Player) {
	// TODO: which stdlib sort guarantees stability?
}
