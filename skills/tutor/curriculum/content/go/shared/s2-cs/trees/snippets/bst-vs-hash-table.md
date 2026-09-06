In Go:

There is no tree in Go's standard library — `map` is the hash table, and when
ordering is needed Go programmers usually sort a slice. So in the exercise you
build the BST yourself, from the same raw material as your linked list —
a struct and pointers, with `nil` as the empty tree:

```go
type Node struct {
	Key   int
	Left  *Node
	Right *Node
}
```
