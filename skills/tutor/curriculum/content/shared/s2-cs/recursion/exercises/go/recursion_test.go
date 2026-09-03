package recursion

import (
	"runtime/debug"
	"testing"
)

func TestFactorial(t *testing.T) {
	cases := []struct {
		n    int
		want int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{5, 120},
		{10, 3628800},
	}
	for _, c := range cases {
		if got := Factorial(c.n); got != c.want {
			t.Errorf("Factorial(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

func TestReverse(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"single rune", "x", "x"},
		{"word", "stack", "kcats"},
		{"multibyte runes", "héllo", "olléh"},
		{"non-latin", "日本語", "語本日"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Reverse(c.in); got != c.want {
				t.Errorf("Reverse(%q) = %q, want %q (reverse runes, not bytes)",
					c.in, got, c.want)
			}
		})
	}
}

// sampleTree: 1 + 2 + 4 + 8 + 3 = 18
//
//	    1
//	   / \
//	  2   3
//	 / \
//	4   8
func sampleTree() *Node {
	return &Node{Value: 1, Children: []*Node{
		{Value: 2, Children: []*Node{
			{Value: 4},
			{Value: 8},
		}},
		{Value: 3},
	}}
}

func sumCases() []struct {
	name string
	root *Node
	want int
} {
	return []struct {
		name string
		root *Node
		want int
	}{
		{"nil tree", nil, 0},
		{"single node", &Node{Value: 7}, 7},
		{"nested tree", sampleTree(), 18},
	}
}

func TestSum(t *testing.T) {
	for _, c := range sumCases() {
		t.Run(c.name, func(t *testing.T) {
			if got := Sum(c.root); got != c.want {
				t.Errorf("Sum(%s) = %d, want %d", c.name, got, c.want)
			}
		})
	}
}

func TestSumIterative(t *testing.T) {
	for _, c := range sumCases() {
		t.Run(c.name, func(t *testing.T) {
			if got := SumIterative(c.root); got != c.want {
				t.Errorf("SumIterative(%s) = %d, want %d", c.name, got, c.want)
			}
		})
	}
}

// deepChain builds a degenerate tree — one child per node, depth levels,
// every value 1 — so the correct sum equals depth.
func deepChain(depth int) *Node {
	root := &Node{Value: 1}
	cur := root
	for i := 1; i < depth; i++ {
		child := &Node{Value: 1}
		cur.Children = []*Node{child}
		cur = child
	}
	return root
}

// A recursive walk needs one call-stack frame per level of depth; an explicit
// stack lives on the heap and doesn't care. This test lowers the stack limit
// so a still-recursive SumIterative fails exactly the way LESSON.md
// describes. If your whole test run aborts with
// "fatal error: stack overflow", that IS the feedback: SumIterative is still
// recursing.
func TestSumIterativeDeepTree(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(16 << 20))
	const depth = 1_000_000
	if got := SumIterative(deepChain(depth)); got != depth {
		t.Errorf("SumIterative(chain of %d nodes) = %d, want %d",
			depth, got, depth)
	}
}
