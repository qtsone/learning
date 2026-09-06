In Go:

```go
// The exercise's cache API. Two S3/S5 habits show up here:
// inject the clock (func() time.Time) so tests control time, and
// keep loader calls outside any lock you hold.
c := cache.New[string, User](10_000, 5*time.Minute, time.Now)
u, err := c.GetOrLoad(id, func() (User, error) {
    return fetchUserFromDB(ctx, id) // runs once per key, however many callers
})
```

In production Go you would reach for `golang.org/x/sync/singleflight` for
request collapsing rather than hand-rolling it — but you are about to
hand-roll it, because you cannot reason about a stampede you have never
collapsed yourself.
