In Go:

The memo cache is a `map[int]int` with the comma-ok idiom from the maps
lesson; the table is a slice from `make`. A recursive helper can carry the
cache and a counter explicitly — no globals needed:

```go
func fibMemo(n int, memo map[int]int, computed *int) int {
	if v, ok := memo[n]; ok {
		return v
	}
	*computed++
	// … compute, store in memo[n], return …
}
```

Two Go-specific traps for the exercise:

- **Overflow is silent.** Go's `int` is 64-bit on your machine; `fib(90)`
  fits, `fib(93)` wraps around into garbage with no error. The tests stop at
  90 deliberately.
- **Pick a safe "impossible" sentinel.** In `MinCoins`, marking impossible
  cells with `math.MaxInt` breaks the moment you compute `best[a-c] + 1` on
  one: max-int plus one wraps to a huge *negative* number that then wins every
  `min`. Use `amount + 1` instead — no real answer can use more than `amount`
  coins (the smallest coin is at least 1), so it is unreachable-large yet
  arithmetic-safe. Translate it to `-1` only when returning.
