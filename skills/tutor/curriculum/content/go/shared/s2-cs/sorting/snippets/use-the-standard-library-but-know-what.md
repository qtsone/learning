In Go:

```go
import (
	"cmp"
	"slices"
)

nums := []int{3, 1, 2}
slices.Sort(nums) // ascending, for ordered types (numbers, strings)

type Player struct {
	Name  string
	Score int
}

// Custom order: score descending, ties broken by name ascending.
slices.SortFunc(players, func(a, b Player) int {
	if c := cmp.Compare(b.Score, a.Score); c != 0 { // b before a: descending
		return c
	}
	return cmp.Compare(a.Name, b.Name)
})
```

The comparison function returns negative when `a` should come first, positive
when `b` should, zero when they're equal — and `cmp.Compare` builds that
three-way answer for any ordered type. Multi-key ordering is just "compare the
first key; on a tie, fall through to the next."
