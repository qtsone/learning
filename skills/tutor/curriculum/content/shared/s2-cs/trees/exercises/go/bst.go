package trees

// Node is one node of a binary search tree of unique int keys.
// A tree is referred to by its root *Node; nil is the empty tree.
type Node struct {
	Key   int
	Left  *Node
	Right *Node
}

// Insert returns the root of the tree after adding key.
// Inserting a key that is already present changes nothing.
// Callers use it as: root = Insert(root, key).
func Insert(root *Node, key int) *Node {
	// TODO: implement per LESSON.md — search for key's spot, attach a new
	// node where you fall off the tree, and re-attach returned subtrees.
	return root
}

// Contains reports whether key is in the tree, walking one
// root-to-leaf path using the BST invariant.
func Contains(root *Node, key int) bool {
	// TODO: implement.
	return false
}

// InOrder returns all keys in in-order (left, node, right).
// On a valid BST the result is ascending.
func InOrder(root *Node) []int {
	// TODO: implement.
	return nil
}

// PreOrder returns all keys in pre-order (node, left, right).
func PreOrder(root *Node) []int {
	// TODO: implement.
	return nil
}

// PostOrder returns all keys in post-order (left, right, node).
func PostOrder(root *Node) []int {
	// TODO: implement.
	return nil
}

// Height returns the number of edges on the longest root-to-leaf
// path: -1 for the empty tree, 0 for a single node.
func Height(root *Node) int {
	// TODO: implement.
	return 0
}
