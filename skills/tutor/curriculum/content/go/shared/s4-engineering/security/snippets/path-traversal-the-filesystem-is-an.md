In Go:

`filepath.IsLocal` answers exactly that question — is this path relative, and
does it stay inside the directory it is joined to?

```go
if !filepath.IsLocal(name) {
	return nil, fmt.Errorf("%w: %q", ErrBadName, name)
}
data, err := os.ReadFile(filepath.Join(dir, name))
```

It rejects `../secret.txt`, `a/../../b`, absolute paths, and (on Windows)
reserved device names. Newer Go versions add `os.Root`, which makes the
operating system itself enforce the boundary for a whole directory subtree —
worth knowing once you serve files seriously.
