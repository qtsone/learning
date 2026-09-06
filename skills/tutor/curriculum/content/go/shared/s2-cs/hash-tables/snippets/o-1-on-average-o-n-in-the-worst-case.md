In Go:

The `map` you have used since S1 is exactly this machinery, hidden behind
syntax. The comma-ok idiom is `Get`, the `delete` builtin is `Delete`, and map
keys must be comparable because the runtime must confirm "same key" after
hashing. Even iteration order being deliberately randomized is this lesson:
bucket order is an artifact of hashing, not something meaningful, and Go
randomizes it so programs cannot accidentally depend on it. The standard
library also ships FNV as [`hash/fnv`](https://pkg.go.dev/hash/fnv) — after
the exercise you can check your implementation against it.
