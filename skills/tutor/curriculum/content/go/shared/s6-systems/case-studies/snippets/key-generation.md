**In Go:** the store's conflict error drives the retry, and `randomKey`
draws from `crypto/rand` — guessable keys would be a security bug, not a
style choice:

```go
func (s *Shortener) Create(ctx context.Context, long string) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		key, err := randomKey(7)
		if err != nil {
			return "", err
		}
		switch err := s.store.InsertIfAbsent(ctx, key, long); {
		case err == nil:
			return key, nil
		case errors.Is(err, ErrKeyExists):
			continue
		default:
			return "", err
		}
	}
	return "", errors.New("shortener: no free key in 5 attempts")
}
```

`InsertIfAbsent` is a port (architecture lesson) — a unique-index violation
in SQL or a conditional put in a KV store is an adapter detail. The bound on
the loop matters: unbounded retry turns a store outage into a spin.
