**In Go:** this is exactly the `len`/`cap` machinery from S1. A slice is a
view onto a backing array; `append` writes into spare capacity when it can
and allocates a bigger backing array when it can't:

```go
xs := make([]int, 0, 4)
for i := 0; i < 10; i++ {
	xs = append(xs, i)
	fmt.Println(len(xs), cap(xs)) // watch cap jump: 4, 8, 16…
}
```

That is also why S1 warned you to keep the result of `append`: after a
growth step, the slice points at a *new* backing array.
