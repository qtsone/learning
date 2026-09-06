In Go:

`golang.org/x/crypto/bcrypt` does all of it — random salt, embedded cost,
constant-time comparison:

```go
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// store string(hash); later:
ok := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
```
