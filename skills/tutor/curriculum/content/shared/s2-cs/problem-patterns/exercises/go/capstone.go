package patterns

// The capstone: no pattern names here. For each function, decide which
// pattern from the stage fits and why — write your justification down
// before coding. Your tutor will ask for it.

// BalancedBrackets reports whether every (, [ and { in s is closed by its
// matching closer in the right order. Runes other than ()[]{} are ignored.
func BalancedBrackets(s string) bool {
	// TODO: which structure hands back the most recently opened bracket first?
	return false
}

// CountIslands returns the number of connected groups of '#' cells in grid,
// where cells connect up/down/left/right (not diagonally) and '.' is water.
// It may mark cells of grid in place while working.
func CountIslands(grid [][]byte) int {
	// TODO: the grid is a graph in disguise — which lesson explored
	// everything reachable from a starting point?
	return 0
}

// MaxNonAdjacentSum returns the largest possible sum of a selection of nums
// (all non-negative) in which no two selected elements are adjacent.
// Selecting nothing is allowed, so an empty slice yields 0.
// Required: O(n) time, O(1) extra space.
func MaxNonAdjacentSum(nums []int) int {
	// TODO: one take-or-skip decision per element — which lesson handles
	// overlapping subproblems?
	return 0
}
