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
	if root == nil {
		return &Node{Key: key}
	}
	switch {
	case key < root.Key:
		root.Left = Insert(root.Left, key)
	case key > root.Key:
		root.Right = Insert(root.Right, key)
	}
	return root
}

// Contains reports whether key is in the tree, walking one
// root-to-leaf path using the BST invariant.
func Contains(root *Node, key int) bool {
	for root != nil {
		switch {
		case key < root.Key:
			root = root.Left
		case key > root.Key:
			root = root.Right
		default:
			return true
		}
	}
	return false
}

// InOrder returns all keys in in-order (left, node, right).
// On a valid BST the result is ascending.
func InOrder(root *Node) []int {
	if root == nil {
		return nil
	}
	keys := append(InOrder(root.Left), root.Key)
	return append(keys, InOrder(root.Right)...)
}

// PreOrder returns all keys in pre-order (node, left, right).
func PreOrder(root *Node) []int {
	if root == nil {
		return nil
	}
	keys := append([]int{root.Key}, PreOrder(root.Left)...)
	return append(keys, PreOrder(root.Right)...)
}

// PostOrder returns all keys in post-order (left, right, node).
func PostOrder(root *Node) []int {
	if root == nil {
		return nil
	}
	keys := append(PostOrder(root.Left), PostOrder(root.Right)...)
	return append(keys, root.Key)
}

// Height returns the number of edges on the longest root-to-leaf
// path: -1 for the empty tree, 0 for a single node.
func Height(root *Node) int {
	if root == nil {
		return -1
	}
	return 1 + max(Height(root.Left), Height(root.Right))
}
