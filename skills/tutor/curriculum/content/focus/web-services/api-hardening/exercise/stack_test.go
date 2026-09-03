package harden

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func newStack(t *testing.T, clk Clock, burst int) http.Handler {
	t.Helper()
	return Harden(http.HandlerFunc(CreateTaskHandler), Options{
		Limiter: NewLimiter(1, burst, clk),
		CORS:    mustCORS(t, testPolicy()),
		Timeout: 5 * time.Second,
	})
}

func post(h http.Handler, addr, origin string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = addr
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

const validBody = `{"title":"ship it","priority":1}`

func TestHardenServesAGoodRequest(t *testing.T) {
	h := newStack(t, newFakeClock(), 5)

	rec := post(h, "192.0.2.10:4000", "https://app.example.com", validBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff on every response", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
}

// Order argument #1: a browser can only show a developer the 429 if the
// rejected response still carries the CORS headers, so CORS must sit OUTSIDE
// the limiter. Security headers must reach every response too.
func TestHardenKeepsHeadersOnRejectedRequests(t *testing.T) {
	h := newStack(t, newFakeClock(), 1)

	if rec := post(h, "192.0.2.11:4000", "https://app.example.com", validBody); rec.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want 201", rec.Code)
	}
	rec := post(h, "192.0.2.11:4000", "https://app.example.com", validBody)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin on the 429 = %q, want the origin: CORS belongs outside RateLimit, "+
			"or the browser turns this into an unexplained network error", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options on the 429 = %q, want nosniff: SecurityHeaders belongs outermost", got)
	}
}

// countingReader records whether anything ever pulled bytes from the body.
type countingReader struct {
	mu    sync.Mutex
	data  *strings.Reader
	reads int
}

func (c *countingReader) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	return c.data.Read(p)
}

func (c *countingReader) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads
}

// Order argument #2: a client over its budget must cost a map lookup, not a
// body read and a JSON decode. If parsing happens before the limiter, an
// attacker gets you to do the expensive work anyway.
func TestHardenRefusesBeforeTouchingTheBody(t *testing.T) {
	h := newStack(t, newFakeClock(), 1)

	if rec := post(h, "192.0.2.12:4000", "", validBody); rec.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want 201", rec.Code)
	}

	body := &countingReader{data: strings.NewReader(validBody)}
	r := httptest.NewRequest(http.MethodPost, "/tasks", body)
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = "192.0.2.12:4000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if n := body.count(); n != 0 {
		t.Errorf("the body was read %d times for a refused request, want 0: rate limiting goes outside parsing", n)
	}
}

func TestHardenRejectsAnOversizedBody(t *testing.T) {
	h := newStack(t, newFakeClock(), 5)

	huge := `{"title":"` + strings.Repeat("a", int(DefaultMaxBodyBytes)+1000) + `","priority":1}`
	rec := post(h, "192.0.2.13:4000", "", huge)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413: body = %s", rec.Code, rec.Body.String())
	}
}

// A preflight is answered by CORS before the limiter sees it. That is a
// deliberate trade: preflights are small and bodyless, and the alternative is
// a browser that reports "network error" instead of your 429.
func TestHardenAnswersPreflightsWithoutSpendingTokens(t *testing.T) {
	h := newStack(t, newFakeClock(), 1)

	for i := 0; i < 3; i++ {
		rec := preflight(t, h, "https://app.example.com", http.MethodPost)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("preflight %d status = %d, want 204", i+1, rec.Code)
		}
	}
	if rec := post(h, "192.0.2.14:4000", "https://app.example.com", validBody); rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201: preflights must not drain the request budget", rec.Code)
	}
}

func TestNewServerSetsEveryTimeout(t *testing.T) {
	h := okHandler(http.StatusOK, "ok")
	srv := NewServer(":8080", h)

	if srv.Addr != ":8080" || srv.Handler == nil {
		t.Fatalf("Addr = %q, Handler set = %v", srv.Addr, srv.Handler != nil)
	}
	for name, d := range map[string]time.Duration{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout,
		"ReadTimeout":       srv.ReadTimeout,
		"WriteTimeout":      srv.WriteTimeout,
		"IdleTimeout":       srv.IdleTimeout,
	} {
		if d <= 0 {
			t.Errorf("%s = %v, want a positive duration: zero means no limit", name, d)
		}
	}
	if srv.ReadHeaderTimeout > srv.ReadTimeout {
		t.Errorf("ReadHeaderTimeout (%v) > ReadTimeout (%v): headers are a prefix of the read", srv.ReadHeaderTimeout, srv.ReadTimeout)
	}
	if srv.WriteTimeout <= HandlerTimeout {
		t.Errorf("WriteTimeout (%v) <= HandlerTimeout (%v): the connection would die before the 503 explaining why",
			srv.WriteTimeout, HandlerTimeout)
	}
	if srv.MaxHeaderBytes <= 0 {
		t.Error("MaxHeaderBytes is unset; set it explicitly so the limit is a decision, not a default")
	}
}

func TestHardenAppliesThePerRequestTimeout(t *testing.T) {
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done(): // the deadline cancels the request context
		case <-time.After(2 * time.Second):
			// Safety net: without a Timeout in the chain nothing ever cancels
			// this context, and a hung test teaches nothing.
		}
	})
	h := Harden(slow, Options{
		Limiter: NewLimiter(1, 5, newFakeClock()),
		// Not a performance assertion: the handler waits for the deadline,
		// however long the race detector makes that take.
		Timeout: 50 * time.Millisecond,
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != timeoutBody {
		t.Errorf("body = %s, want %s", got, timeoutBody)
	}
}
