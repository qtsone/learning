package patterns

// BalancedBrackets reports whether every (, [ and { in s is closed by its
// matching closer in the right order. Runes other than ()[]{} are ignored.
//
// Pattern: stack — nesting means the most recently opened bracket must be
// the first one closed.
func BalancedBrackets(s string) bool {
	pairs := map[rune]rune{')': '(', ']': '[', '}': '{'}
	var stack []rune
	for _, r := range s {
		switch r {
		case '(', '[', '{':
			stack = append(stack, r)
		case ')', ']', '}':
			if len(stack) == 0 || stack[len(stack)-1] != pairs[r] {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}
	return len(stack) == 0
}

// CountIslands returns the number of connected groups of '#' cells in grid,
// where cells connect up/down/left/right (not diagonally) and '.' is water.
// It may mark cells of grid in place while working.
//
// Pattern: graph flood fill — each unvisited land cell starts a DFS (with an
// explicit stack, so depth is bounded by memory, not the call stack) that
// sinks the whole island so it is counted exactly once. O(rows*cols).
func CountIslands(grid [][]byte) int {
	type cell struct{ r, c int }
	dirs := [4]cell{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	count := 0
	var stack []cell
	for r := range grid {
		for c := range grid[r] {
			if grid[r][c] != '#' {
				continue
			}
			count++
			grid[r][c] = '.'
			stack = append(stack[:0], cell{r, c})
			for len(stack) > 0 {
				cur := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				for _, d := range dirs {
					nr, nc := cur.r+d.r, cur.c+d.c
					if nr < 0 || nr >= len(grid) || nc < 0 || nc >= len(grid[nr]) {
						continue
					}
					if grid[nr][nc] == '#' {
						grid[nr][nc] = '.'
						stack = append(stack, cell{nr, nc})
					}
				}
			}
		}
	}
	return count
}

// MaxNonAdjacentSum returns the largest possible sum of a selection of nums
// (all non-negative) in which no two selected elements are adjacent.
// Selecting nothing is allowed, so an empty slice yields 0.
// Required: O(n) time, O(1) extra space.
//
// Pattern: dynamic programming — one take-or-skip decision per element, and
// only the previous two results feed the next decision, so two rolling
// variables replace the whole table.
func MaxNonAdjacentSum(nums []int) int {
	// incl: best sum for the prefix when the latest element is taken.
	// excl: best sum when it is skipped.
	incl, excl := 0, 0
	for _, v := range nums {
		incl, excl = excl+v, max(incl, excl)
	}
	return max(incl, excl)
}
