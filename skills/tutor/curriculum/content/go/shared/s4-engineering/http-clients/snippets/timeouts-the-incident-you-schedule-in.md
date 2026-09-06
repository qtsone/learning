In Go:

```go
client := &http.Client{Timeout: 10 * time.Second} // ceiling for everything

ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
defer cancel()
req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
```

You met `context` in the concurrency arc — this is its natural habitat.
`http.NewRequestWithContext` ties the request to the context: when the
deadline passes or the caller cancels, the transport abandons the request and
`Do` returns an error wrapping `context.DeadlineExceeded` or
`context.Canceled`. And never reach for `http.DefaultClient` in production
code: it is the shared, zero-timeout client.
