**In Go:** `int` is 64 bits on mainstream platforms, so a slice large enough
to overflow `lo + hi` won't fit in memory — but write the safe form anyway.
It costs nothing, it is correct in every language and on 32-bit targets, and
reviewers recognize it as the hand of someone who knows the history.

```go
for lo < hi {
	mid := lo + (hi-lo)/2
	switch {
	case xs[mid] == target:
		return mid
	case xs[mid] < target:
		lo = mid + 1
	default:
		hi = mid
	}
}
```
