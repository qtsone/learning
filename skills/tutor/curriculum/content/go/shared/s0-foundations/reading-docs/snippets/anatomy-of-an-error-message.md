**In Go:** the compiler reports errors as `file:line:column`, a precise home
address for the mistake:

```
./main.go:7:2: undefined: fmt.Printlnn
```

File `main.go`, line 7, column 2: there is no such name as `fmt.Printlnn` (a
typo for `fmt.Println`). You will read hundreds of these in the next stage.
