package harden

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// fakeClock is the injected clock every time-dependent test uses. Nothing in
// this package may call time.Now directly, so no test ever sleeps and no test
// depends on how loaded the machine is. It is mutex-guarded because the
// concurrency test reads it from many goroutines at once.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// unreachable fails the test if it is ever served: middleware that rejects a
// request must not call the next handler.
func unreachable(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("the wrapped handler ran for a request that should have been rejected: %s %s", r.Method, r.URL.Path)
	})
}

func okHandler(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

// envelope is the decoded error body every failure path must produce.
type envelope struct {
	Error struct {
		Message string       `json:"message"`
		Fields  []FieldError `json:"fields"`
	} `json:"error"`
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("response body %q is not the JSON error envelope: %v", rec.Body.String(), err)
	}
	if env.Error.Message == "" {
		t.Errorf("error envelope has an empty message: %s", rec.Body.String())
	}
	return env
}
