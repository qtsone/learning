package trees

import (
	"slices"
	"testing"
)

func build(keys ...int) *Node {
	var root *Node
	for _, k := range keys {
		root = Insert(root, k)
	}
	return root
}

// balancedOrder appends the keys of sorted in an insertion order
// (medians first) that builds a perfectly balanced BST.
func balancedOrder(sorted []int, out []int) []int {
	if len(sorted) == 0 {
		return out
	}
	mid := len(sorted) / 2
	out = append(out, sorted[mid])
	out = balancedOrder(sorted[:mid], out)
	return balancedOrder(sorted[mid+1:], out)
}

func TestInsertBuildsLinkedStructure(t *testing.T) {
	root := build(5, 3, 8)
	if root == nil {
		t.Fatal("Insert(nil, 5) returned nil; want a new root node")
	}
	if root.Key != 5 {
		t.Fatalf("root.Key = %d, want 5 (first inserted key stays the root)", root.Key)
	}
	if root.Left == nil || root.Left.Key != 3 {
		t.Errorf("root.Left = %+v, want node with Key 3 (3 < 5 goes left)", root.Left)
	}
	if root.Right == nil || root.Right.Key != 8 {
		t.Errorf("root.Right = %+v, want node with Key 8 (8 > 5 goes right)", root.Right)
	}
}

func TestContains(t *testing.T) {
	root := build(5, 3, 8, 1, 4, 7, 9)
	cases := []struct {
		name string
		key  int
		want bool
	}{
		{"root", 5, true},
		{"leaf", 4, true},
		{"inner node", 8, true},
		{"absent between keys", 6, false},
		{"absent below min", 0, false},
		{"absent above max", 100, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Contains(root, c.key); got != c.want {
				t.Errorf("Contains(root, %d) = %v, want %v", c.key, got, c.want)
			}
		})
	}
	if Contains(nil, 5) {
		t.Error("Contains(nil, 5) = true, want false (empty tree holds nothing)")
	}
}

func TestInsertIgnoresDuplicates(t *testing.T) {
	root := build(5, 3, 8, 3, 5, 8)
	if got := len(InOrder(root)); got != 3 {
		t.Errorf("tree holds %d keys after re-inserting duplicates, want 3 (duplicates change nothing)", got)
	}
}

func TestInOrderIsSorted(t *testing.T) {
	shuffled := []int{42, 7, 93, 15, 68, 3, 27, 81, 50, 12, 99, 34, 61, 5, 76, 20, 88, 46, 9, 70}
	root := build(shuffled...)
	got := InOrder(root)
	want := slices.Clone(shuffled)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("InOrder = %v, want ascending %v (left, node, right on a BST emits sorted order)", got, want)
	}
}

func TestTraversalOrders(t *testing.T) {
	root := build(5, 3, 8, 1, 4, 7, 9)
	cases := []struct {
		name     string
		traverse func(*Node) []int
		want     []int
	}{
		{"InOrder", InOrder, []int{1, 3, 4, 5, 7, 8, 9}},
		{"PreOrder", PreOrder, []int{5, 3, 1, 4, 8, 7, 9}},
		{"PostOrder", PostOrder, []int{1, 4, 3, 7, 9, 8, 5}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.traverse(root); !slices.Equal(got, c.want) {
				t.Errorf("%s = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestHeight(t *testing.T) {
	cases := []struct {
		name string
		keys []int
		want int
	}{
		{"empty tree", nil, -1},
		{"single node", []int{5}, 0},
		{"balanced 7 keys", []int{5, 3, 8, 1, 4, 7, 9}, 2},
		{"ascending chain of 10", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 9},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Height(build(c.keys...)); got != c.want {
				t.Errorf("Height = %d, want %d", got, c.want)
			}
		})
	}
}

// Same 1023 keys, two insertion orders: sorted input degrades the tree
// to a chain (height n-1, operations O(n)); median-first input keeps it
// balanced (height log2(n), operations O(log n)).
func TestInsertionOrderDecidesHeight(t *testing.T) {
	const n = 1023
	sorted := make([]int, n)
	for i := range sorted {
		sorted[i] = i
	}

	chain := build(sorted...)
	if got, want := Height(chain), n-1; got != want {
		t.Errorf("height after sorted insertion = %d, want %d (every key hangs off the right: a chain)", got, want)
	}

	balanced := build(balancedOrder(sorted, nil)...)
	if got, want := Height(balanced), 9; got != want {
		t.Errorf("height after median-first insertion = %d, want %d (1023 keys fit in 10 full levels)", got, want)
	}
}
