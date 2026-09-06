In Go:

```go
key := os.Getenv("VAULT_API_KEY")
if key == "" {
	return "", errors.New("VAULT_API_KEY is not set")
}
```
