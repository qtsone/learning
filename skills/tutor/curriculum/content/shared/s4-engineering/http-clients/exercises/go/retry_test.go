package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 5, BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, time.Second},
		{10, time.Second},
	}
	for _, tc := range cases {
		if got := Backoff(p, tc.attempt); got != tc.want {
			t.Errorf("Backoff(attempt=%d) = %v, want %v (double per attempt, cap at MaxDelay)",
				tc.attempt, got, tc.want)
		}
	}
}

func TestJitterStaysInRangeAndVaries(t *testing.T) {
	const d = 100 * time.Millisecond
	seen := map[time.Duration]bool{}
	for i := 0; i < 200; i++ {
		got := Jitter(d)
		if got < 0 || got > d {
			t.Fatalf("Jitter(%v) = %v, want a value in [0, %v]", d, got, d)
		}
		seen[got] = true
	}
	if len(seen) < 2 {
		t.Error("Jitter returned the same value 200 times — it must be random, or every client retries in lockstep")
	}
	if got := Jitter(0); got != 0 {
		t.Errorf("Jitter(0) = %v, want 0", got)
	}
}

func TestShouldRetry(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"transport failure", errors.New("dial tcp: connection refused"), true},
		{"server error 500", &APIError{StatusCode: 500}, true},
		{"bad gateway 502", &APIError{StatusCode: 502}, true},
		{"unavailable 503", &APIError{StatusCode: 503}, true},
		{"rate limited 429", &APIError{StatusCode: 429}, true},
		{"bad request 400", &APIError{StatusCode: 400}, false},
		{"not found 404", &APIError{StatusCode: 404}, false},
		{"forbidden 403", &APIError{StatusCode: 403}, false},
		{"wrapped api error", fmt.Errorf("get widgets: %w", &APIError{StatusCode: 503}), true},
		{"context cancelled", context.Canceled, false},
		{"wrapped deadline exceeded", fmt.Errorf("do request: %w", context.DeadlineExceeded), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldRetry(tc.err); got != tc.want {
				t.Errorf("ShouldRetry(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

func TestGetJSONRetryEventualSuccess(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= 2 {
			http.Error(w, "temporarily overloaded", http.StatusServiceUnavailable)
			return
		}
		io.WriteString(w, `{"name":"go","major":1}`)
	}))
	defer ts.Close()

	c := New(ts.URL)
	var waits []time.Duration
	c.sleep = func(d time.Duration) { waits = append(waits, d) }
	p := RetryPolicy{MaxAttempts: 5, BaseDelay: 50 * time.Millisecond, MaxDelay: 400 * time.Millisecond}

	var rel release
	if err := c.GetJSONRetry(context.Background(), "/", &rel, p); err != nil {
		t.Fatalf("GetJSONRetry = %v, want success after two 503s", err)
	}
	if rel.Name != "go" {
		t.Errorf("decoded %+v, want the successful response decoded", rel)
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("server saw %d requests, want 3 (two failures, then success)", got)
	}
	if len(waits) != 2 {
		t.Fatalf("recorded %d retry waits, want 2 — wait via c.sleep between attempts only", len(waits))
	}
	for i, d := range waits {
		bound := Backoff(p, i)
		if d < 0 || d > bound {
			t.Errorf("wait %d = %v, want Jitter(Backoff) in [0, %v]", i, d, bound)
		}
	}
}

func TestGetJSONRetryDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "no such widget", http.StatusNotFound)
	}))
	defer ts.Close()

	c := New(ts.URL)
	var waits []time.Duration
	c.sleep = func(d time.Duration) { waits = append(waits, d) }

	var rel release
	err := c.GetJSONRetry(context.Background(), "/", &rel,
		RetryPolicy{MaxAttempts: 4, BaseDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("GetJSONRetry = %v, want the 404 *APIError surfaced immediately", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server saw %d requests, want 1 — a 404 will fail identically every time, do not retry it", got)
	}
	if len(waits) != 0 {
		t.Errorf("recorded %d waits, want 0 for a non-retryable failure", len(waits))
	}
}

func TestGetJSONRetryGivesUpAfterMaxAttempts(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "still down", http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	c := New(ts.URL)
	var waits []time.Duration
	c.sleep = func(d time.Duration) { waits = append(waits, d) }

	var rel release
	err := c.GetJSONRetry(context.Background(), "/", &rel,
		RetryPolicy{MaxAttempts: 3, BaseDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GetJSONRetry = %v, want the final 503 *APIError after giving up", err)
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("server saw %d requests, want exactly MaxAttempts=3", got)
	}
	if len(waits) != 2 {
		t.Errorf("recorded %d waits, want 2 — no wait after the final attempt", len(waits))
	}
}

func TestGetJSONRetryHonorsCancelledContext(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		io.WriteString(w, `{"name":"go","major":1}`)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := New(ts.URL)
	var rel release
	err := c.GetJSONRetry(ctx, "/", &rel,
		RetryPolicy{MaxAttempts: 3, BaseDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetJSONRetry = %v, want errors.Is(err, context.Canceled)", err)
	}
	if got := calls.Load(); got != 0 {
		t.Errorf("server saw %d requests, want 0 — the caller already gave up", got)
	}
}
