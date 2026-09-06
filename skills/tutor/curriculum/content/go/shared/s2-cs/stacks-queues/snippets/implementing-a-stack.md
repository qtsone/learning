In Go: a slice plus `append` is already 90% of a stack — this is why Go has
no stack type in the standard library.

```go
s.items = append(s.items, r)          // push
top := s.items[len(s.items)-1]        // read the top
s.items = s.items[:len(s.items)-1]    // shrink by one: pop's second half
```
