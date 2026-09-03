package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// statusServer answers /ok with 200, /gone with 404, /boom with 500 and
// /empty with 204. Tests never touch the real network: httptest listens on
// a 127.0.0.1 port owned by this test process.
func statusServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/gone", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/boom", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/empty", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestCheckReportsStatusesInOrder(t *testing.T) {
	srv := statusServer(t)
	urls := []string{
		srv.URL + "/ok",
		srv.URL + "/gone",
		srv.URL + "/ok",
		srv.URL + "/boom",
		srv.URL + "/empty",
	}
	wantStatus := []int{200, 404, 200, 500, 204}

	c := &Checker{Client: srv.Client(), Concurrency: 3, Timeout: 5 * time.Second}
	results := c.Check(context.Background(), urls)

	if len(results) != len(urls) {
		t.Fatalf("Check returned %d results, want %d (one per URL)", len(results), len(urls))
	}
	for i, r := range results {
		if r.URL != urls[i] {
			t.Errorf("results[%d].URL = %q, want %q (results must keep input order)", i, r.URL, urls[i])
			continue
		}
		if r.Err != nil {
			t.Errorf("results[%d] (%s): unexpected error: %v", i, r.URL, r.Err)
			continue
		}
		if r.Status != wantStatus[i] {
			t.Errorf("results[%d] (%s): Status = %d, want %d", i, r.URL, r.Status, wantStatus[i])
		}
	}
}

func TestCheckReportsTransportErrors(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	deadURL := srv.URL
	srv.Close() // nothing listens there anymore

	c := &Checker{Concurrency: 2, Timeout: 5 * time.Second}
	results := c.Check(context.Background(), []string{deadURL, "::not-a-url"})

	if len(results) != 2 {
		t.Fatalf("Check returned %d results, want 2", len(results))
	}
	if results[0].Err == nil {
		t.Errorf("checking a dead server: Err = nil, want a connection error")
	}
	if results[1].Err == nil {
		t.Errorf("checking a malformed URL: Err = nil, want an error")
	}
}

func TestCheckBoundsConcurrency(t *testing.T) {
	const limit = 3
	const total = 8

	var mu sync.Mutex
	inFlight, maxInFlight := 0, 0
	arrived := make(chan struct{}, total)
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()
		arrived <- struct{}{}
		<-release // hold the request open until the test lets go
		mu.Lock()
		inFlight--
		mu.Unlock()
	}))
	defer srv.Close()
	defer close(release) // unblock any stragglers so srv.Close can finish

	urls := make([]string, total)
	for i := range urls {
		urls[i] = fmt.Sprintf("%s/%d", srv.URL, i)
	}
	c := &Checker{Client: srv.Client(), Concurrency: limit, Timeout: time.Minute}

	done := make(chan []Result, 1)
	go func() { done <- c.Check(context.Background(), urls) }()

	watchdog := time.After(10 * time.Second)

	// Phase 1: with 8 jobs and a pool of 3, exactly 3 requests must be in
	// flight together before anything is released.
	for i := 0; i < limit; i++ {
		select {
		case <-arrived:
		case results := <-done:
			t.Fatalf("Check returned %d results before any request was answered — it must wait for its requests", len(results))
		case <-watchdog:
			t.Fatalf("timed out waiting for %d simultaneous requests — is Check actually running checks concurrently?", limit)
		}
	}

	// Phase 2: release one held request per remaining job; each freed slot
	// must admit the next URL.
	for i := 0; i < total-limit; i++ {
		release <- struct{}{}
		select {
		case <-arrived:
		case <-watchdog:
			t.Fatal("timed out waiting for the next request after a slot freed — are workers reusing their slot?")
		}
	}
	for i := 0; i < limit; i++ {
		release <- struct{}{}
	}

	select {
	case results := <-done:
		if len(results) != total {
			t.Fatalf("Check returned %d results, want %d", len(results), total)
		}
	case <-watchdog:
		t.Fatal("timed out waiting for Check to return after all requests were released")
	}

	mu.Lock()
	got := maxInFlight
	mu.Unlock()
	if got > limit {
		t.Errorf("saw %d requests in flight at once, want at most %d (Concurrency)", got, limit)
	}
}

func TestCheckAppliesPerRequestTimeout(t *testing.T) {
	block := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/fast", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		<-block // never answers within the test
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	defer close(block)

	urls := []string{srv.URL + "/fast", srv.URL + "/slow", srv.URL + "/fast"}
	c := &Checker{Client: srv.Client(), Concurrency: 3, Timeout: 100 * time.Millisecond}
	results := c.Check(context.Background(), urls)

	if len(results) != len(urls) {
		t.Fatalf("Check returned %d results, want %d", len(results), len(urls))
	}
	for _, i := range []int{0, 2} {
		if results[i].Err != nil {
			t.Errorf("results[%d] (fast URL): unexpected error: %v — one slow URL must not fail the others", i, results[i].Err)
		}
	}
	if results[1].Err == nil {
		t.Fatalf("results[1] (stuck URL): Err = nil, want a timeout error")
	}
	if !errors.Is(results[1].Err, context.DeadlineExceeded) {
		t.Errorf("results[1].Err = %v, want it to wrap context.DeadlineExceeded (per-request context.WithTimeout)", results[1].Err)
	}
}

func TestCheckCancelReportsPartialResults(t *testing.T) {
	const limit = 2
	const total = 5

	var hits atomic.Int64
	arrived := make(chan struct{}, total)
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		arrived <- struct{}{}
		<-release // held open until test cleanup
	}))
	defer srv.Close()
	defer close(release)

	urls := make([]string, total)
	for i := range urls {
		urls[i] = fmt.Sprintf("%s/%d", srv.URL, i)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &Checker{Client: srv.Client(), Concurrency: limit, Timeout: time.Minute}

	done := make(chan []Result, 1)
	go func() { done <- c.Check(ctx, urls) }()

	watchdog := time.After(10 * time.Second)

	// Wait until both workers are provably in flight, then pull the plug.
	for i := 0; i < limit; i++ {
		select {
		case <-arrived:
		case results := <-done:
			t.Fatalf("Check returned %d results before any request was answered", len(results))
		case <-watchdog:
			t.Fatal("timed out waiting for the first in-flight requests")
		}
	}
	cancel()

	var results []Result
	select {
	case results = <-done:
	case <-watchdog:
		t.Fatal("Check did not return after cancellation — in-flight requests must be aborted, not awaited")
	}

	if len(results) != total {
		t.Fatalf("got %d results after cancel, want %d — report every URL, the unstarted ones as canceled", len(results), total)
	}
	for i, r := range results {
		if r.URL != urls[i] {
			t.Errorf("results[%d].URL = %q, want %q (results must keep input order)", i, r.URL, urls[i])
		}
		if r.Err == nil {
			t.Errorf("results[%d]: Err = nil after cancellation, want an error wrapping context.Canceled", i)
			continue
		}
		if !errors.Is(r.Err, context.Canceled) {
			t.Errorf("results[%d].Err = %v, want it to wrap context.Canceled", i, r.Err)
		}
	}
	if got := hits.Load(); got > limit {
		t.Errorf("server received %d requests, want at most %d — no new requests may start after cancellation", got, limit)
	}
}
