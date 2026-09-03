package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// testCost is bcrypt.MinCost. The work factor is a production knob, not
	// something these tests check, and a slow suite is a suite nobody runs.
	testCost = bcrypt.MinCost

	testTTL      = 30 * time.Minute
	testPassword = "correct horse battery staple"
)

// testStart is a fixed instant: every time assertion in this suite is relative
// to it, so nothing depends on when the tests run.
var testStart = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// fakeClock is the S5 pattern: time only moves when a test says so. The mutex
// is not decoration — the concurrency test reads it from many goroutines.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock { return &fakeClock{now: testStart} }

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

// countingHasher wraps a real hasher and counts verifications, which is how a
// test can assert that a failed login did the same work as a successful one
// without ever measuring wall-clock time.
type countingHasher struct {
	inner PasswordHasher

	mu       sync.Mutex
	verifies int
}

func (h *countingHasher) Hash(password string) (string, error) { return h.inner.Hash(password) }

func (h *countingHasher) Verify(encoded, password string) error {
	h.mu.Lock()
	h.verifies++
	h.mu.Unlock()
	return h.inner.Verify(encoded, password)
}

func (h *countingHasher) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.verifies
}

type env struct {
	t      *testing.T
	svc    *Service
	clock  *fakeClock
	hasher *countingHasher
	h      http.Handler
}

func newEnv(t *testing.T) *env {
	t.Helper()
	clock := newFakeClock()
	hasher := &countingHasher{inner: Hasher{Cost: testCost}}
	svc := NewService(Config{
		Clock:        clock,
		Hasher:       hasher,
		SessionTTL:   testTTL,
		CookieSecure: true,
	})
	return &env{t: t, svc: svc, clock: clock, hasher: hasher, h: svc.Routes()}
}

func (e *env) register(username, password string) User {
	e.t.Helper()
	u, err := e.svc.Register(username, password)
	if err != nil {
		e.t.Fatalf("Register(%q, …) = error %v, want nil", username, err)
	}
	return u
}

func (e *env) do(req *http.Request, cookies ...*http.Cookie) *http.Response {
	e.t.Helper()
	for _, c := range cookies {
		if c != nil {
			req.AddCookie(c)
		}
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec.Result()
}

func (e *env) post(path, body string, cookies ...*http.Cookie) *http.Response {
	e.t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return e.do(req, cookies...)
}

func (e *env) get(path string, cookies ...*http.Cookie) *http.Response {
	e.t.Helper()
	return e.do(httptest.NewRequest(http.MethodGet, path, nil), cookies...)
}

func (e *env) login(username, password string, cookies ...*http.Cookie) *http.Response {
	e.t.Helper()
	return e.post("/login", fmt.Sprintf(`{"username":%q,"password":%q}`, username, password), cookies...)
}

func readBody(t *testing.T, res *http.Response) []byte {
	t.Helper()
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return b
}

func decodeUser(t *testing.T, body []byte) User {
	t.Helper()
	var envelope struct {
		Data User `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("response %q is not a {\"data\": …} envelope: %v", body, err)
	}
	return envelope.Data
}

func errorMessage(t *testing.T, body []byte) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("response %q is not an {\"error\": {\"message\": …}} envelope: %v", body, err)
	}
	return envelope.Error.Message
}

func sessionCookie(t *testing.T, res *http.Response) *http.Cookie {
	t.Helper()
	c := findCookie(res, SessionCookieName)
	if c == nil {
		t.Fatalf("response has no %q cookie (Set-Cookie: %q)", SessionCookieName, res.Header.Values("Set-Cookie"))
	}
	return c
}

func findCookie(res *http.Response, name string) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}
