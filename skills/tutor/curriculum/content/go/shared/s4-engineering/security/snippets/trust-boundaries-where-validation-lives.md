In Go:

Remember from the strings lesson that indexing a string yields *bytes*. For an
ASCII-only allowlist that is exactly what you want — checking bytes means a
sneaky multi-byte character can never pass as a letter:

```go
for i := 0; i < len(name); i++ {
	c := name[i]
	if c < 'a' || c > 'z' { // plus whatever else the allowlist admits
		return fmt.Errorf("forbidden byte %q at index %d", c, i)
	}
}
```
