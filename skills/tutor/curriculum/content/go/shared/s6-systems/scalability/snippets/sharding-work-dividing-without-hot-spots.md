In Go:

```go
// The three pieces you build, sketched. Note the injected clock in the
// limiter — same testing habit as the caching and message-queue lessons.
tb := NewTokenBucket(20, 5, time.Now) // burst 20, sustained 5/s
if !tb.Allow() {
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(tb.RetryAfter().Seconds()))))
	http.Error(w, "rate limited", http.StatusTooManyRequests)
	return
}

if !queue.Offer(job) { // bounded: full means shed, never grow
	http.Error(w, "overloaded", http.StatusServiceUnavailable)
	return
}

owner, _ := ring.Get(job.Key) // consistent hashing decides which worker
```
