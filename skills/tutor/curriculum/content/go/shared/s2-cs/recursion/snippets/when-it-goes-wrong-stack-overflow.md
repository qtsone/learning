> **In Go:** each goroutine's stack starts tiny and grows on demand, but only
> up to a hard cap (about 1 GB on 64-bit machines by default). Blowing it is
> not a recoverable panic — the program dies with:
>
> ```
> runtime: goroutine stack exceeds 1000000000-byte limit
> fatal error: stack overflow
> ```
>
> You will meet this exact message in the exercise, on purpose.
