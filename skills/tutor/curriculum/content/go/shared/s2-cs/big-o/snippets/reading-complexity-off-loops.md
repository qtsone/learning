In Go, rule 3's shape looks like this — the pair-comparison loop you have
already written by instinct:

```go
for i := 0; i < len(xs); i++ {
	for j := i + 1; j < len(xs); j++ {
		if xs[i] == xs[j] {
			return true // early exit — but the worst case still visits every pair
		}
	}
}
```

The `return true` does not change the class: Big-O rates the worst case, and
the worst case (no duplicates) runs the full n²/2 comparisons.
