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
	// TODO: base case first, then n * Factorial(n-1).
	return 0
}

// Reverse returns s reversed by runes ("héllo" → "olléh"), computed
// recursively. Convert to []rune first — slicing a string slices bytes.
func Reverse(s string) string {
	// TODO: base case: 0 or 1 runes are already reversed.
	return ""
}

// Sum returns the total of every Value in the tree rooted at root,
// computed recursively. Sum(nil) is 0.
func Sum(root *Node) int {
	// TODO: base case for a missing node, one recursive call per child.
	return 0
}

// SumIterative returns the same total as Sum with no recursion: a loop and
// an explicit stack (a slice of *Node) in place of the call stack.
func SumIterative(root *Node) int {
	// TODO: push root, then loop: pop a node, add its value, push its children.
	return 0
}
