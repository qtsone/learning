In Go: you rarely need a mocking framework. Declare a small interface where
the dependency is *consumed* (remember S3 — interfaces belong to the
consumer), and hand-roll the double in the test file:

```go
type stubClock struct{ now time.Time }

func (c stubClock) Now() time.Time { return c.now }

type spyNotifier struct{ messages []string }

func (s *spyNotifier) Notify(m string) error {
	s.messages = append(s.messages, m)
	return nil
}
```

Ten lines, no dependencies, and the double's behavior is in plain sight
inside the test that uses it. This is idiomatic Go testing, and it only works
because the interfaces are small — one more reason to keep them that way.
