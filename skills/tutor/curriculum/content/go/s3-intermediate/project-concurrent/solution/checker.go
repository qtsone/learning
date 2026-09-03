package main

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

// Result is the outcome of checking one URL. Exactly one of the two failure
// signals is meaningful: a completed request records Status (even 404 or
// 500 — the request itself succeeded); a request that never completed
// (malformed URL, connection refused, timeout, canceled) records Err.
type Result struct {
	URL    string
	Status int
	Err    error
}

// Checker checks URLs concurrently.
//
// Client is the HTTP client to use; nil means http.DefaultClient.
// Concurrency is the maximum number of requests in flight at once;
// values <= 0 behave as 1. Timeout bounds each individual request
// (derived per request with context.WithTimeout); <= 0 means no timeout.
type Checker struct {
	Client      *http.Client
	Concurrency int
	Timeout     time.Duration
}

// Check fetches every URL with a bounded worker pool and returns one Result
// per URL, in input order. Canceling ctx aborts in-flight requests and
// starts no new ones; Check still returns a full slice in which the
// affected URLs carry the context's error.
func (c *Checker) Check(ctx context.Context, urls []string) []Result {
	workers := c.Concurrency
	if workers < 1 {
		workers = 1
	}

	// Every index is written by exactly one goroutine and wg.Wait orders
	// all writes before the return, so no further synchronization is
	// needed — and input order is preserved for free.
	results := make([]Result, len(urls))

	// All jobs are known up front, so fill and close the channel before
	// any worker starts; the close is the workers' shutdown signal.
	jobs := make(chan int, len(urls))
	for i := range urls {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if err := ctx.Err(); err != nil {
					results[i] = Result{URL: urls[i], Err: err}
					continue
				}
				results[i] = c.checkOne(ctx, urls[i])
			}
		}()
	}
	wg.Wait()
	return results
}

// checkOne performs a single GET with its own timeout budget derived from
// ctx, so one slow target cannot spend the other URLs' time.
func (c *Checker) checkOne(ctx context.Context, url string) Result {
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{URL: url, Err: err}
	}

	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{URL: url, Err: err}
	}
	defer resp.Body.Close()
	// Drain the body so the underlying connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)

	return Result{URL: url, Status: resp.StatusCode}
}
