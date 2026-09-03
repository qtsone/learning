package main

import (
	"context"
	"net/http"
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
	// TODO: implement the worker pool per the acceptance criteria in
	// LESSON.md — jobs channel, Concurrency workers, results by index,
	// WaitGroup to close over the whole thing.
	return nil
}
