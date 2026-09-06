In Go: `make(map[int]bool)` and `append` that grows a slice allocate memory
proportional to what you put in them — that's auxiliary space. Indexing
`xs[i]` or slicing `xs[2:5]` allocates nothing: a slice expression shares the
backing array (remember the S1 slices lesson). The Go test helper
`testing.AllocsPerRun` counts allocations, which makes "O(1) auxiliary
space" something a test can actually check — the exercise uses it.
