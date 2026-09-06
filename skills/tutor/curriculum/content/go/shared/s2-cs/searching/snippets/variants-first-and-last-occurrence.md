**In Go:** the standard library ships both flavors, and both return the
*boundary*, not "any match":

```go
i, found := slices.BinarySearch(xs, target)      // i = lower bound; found = presence
j := sort.Search(len(xs), func(i int) bool {     // first i where the predicate
	return xs[i] >= target                       // flips to true
})
```
