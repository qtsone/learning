package recursion

// A Node is a value with zero or more child nodes — the shape of anything
// nested to unknown depth: folders in folders, comments and their replies.
// Children never contains nil.
type Node struct {
	Value    int
	Children []*Node
}

// Factorial returns n! for n >= 0, computed recursively. Factorial(0) is 1.
func Factorial(n int) int {
	if n == 0 {
		return 1
	}
	return n * Factorial(n-1)
}

// Reverse returns s reversed by runes ("héllo" → "olléh"), computed
// recursively. Convert to []rune first — slicing a string slices bytes.
func Reverse(s string) string {
	rs := []rune(s)
	reverseRunes(rs)
	return string(rs)
}

// reverseRunes swaps the outermost pair in place, then recurses on what is
// between them. Zero or one runes are already reversed.
func reverseRunes(rs []rune) {
	if len(rs) < 2 {
		return
	}
	rs[0], rs[len(rs)-1] = rs[len(rs)-1], rs[0]
	reverseRunes(rs[1 : len(rs)-1])
}

// Sum returns the total of every Value in the tree rooted at root,
// computed recursively. Sum(nil) is 0.
func Sum(root *Node) int {
	if root == nil {
		return 0
	}
	total := root.Value
	for _, child := range root.Children {
		total += Sum(child)
	}
	return total
}

// SumIterative returns the same total as Sum with no recursion: a loop and
// an explicit stack (a slice of *Node) in place of the call stack. The slice
// lives on the heap, so depth is bounded by memory, not by the goroutine
// stack limit.
func SumIterative(root *Node) int {
	if root == nil {
		return 0
	}
	total := 0
	stack := []*Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		total += n.Value
		stack = append(stack, n.Children...)
	}
	return total
}
