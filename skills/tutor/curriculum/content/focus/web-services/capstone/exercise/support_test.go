package board

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// hangGuard bounds how long a test waits for something that should already have
// happened. It is a guard against a hung suite, never an assertion about how
// fast your code is: every timing rule in this service is driven by the
// injected clock, and no test sleeps to let time pass.
const hangGuard = 2 * time.Second

// testStart is the fixed instant the fake clock starts at, so heartbeats,
// timestamps and therefore ETags are the same on every machine on every day.
var testStart = time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)

// testPassword is the one password every fixture account has.
const testPassword = "correct-horse-battery"

// fakeClock is the S5 pattern extended with a ticker a test fires by hand.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	tickers chan chan time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: testStart, tickers: make(chan chan time.Time, 4)}
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

// NewTicker hands the handler an unbuffered channel and keeps a reference so a
// test can deliver a tick to it. Unbuffered is what makes a tick a
// synchronisation point instead of a race.
func (c *fakeClock) NewTicker(time.Duration) (<-chan time.Time, func()) {
	ch := make(chan time.Time)
	select {
	case c.tickers <- ch:
	default:
	}
	return ch, func() {}
}

// tick delivers one heartbeat tick and waits until the handler takes it.
func (c *fakeClock) tick(t *testing.T) {
	t.Helper()
	var ch chan time.Time
	select {
	case ch = <-c.tickers:
	case <-time.After(hangGuard):
		t.Fatal("no ticker was created from the injected Clock: heartbeats must come from Clock.NewTicker, not time.Tick")
	}
	select {
	case ch <- c.Now():
	case <-time.After(hangGuard):
		t.Fatal("the tick was never received: the streaming loop must select on the ticker channel")
	}
	select { // put it back so a later tick reaches the same handler
	case c.tickers <- ch:
	default:
	}
}

// quietLogger keeps expected denials and failures out of the test output.
func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// env is one whole service, wired the way main would wire it, with a fake clock
// and four fixture accounts.
type env struct {
	t       *testing.T
	srv     *Server
	store   *Store
	hub     *Hub
	clock   *fakeClock
	handler http.Handler
	pool    *Pool
	users   map[string]User
	seq     int
}

func newEnv(t *testing.T) *env { return newEnvWith(t, nil) }

// newEnvWith builds the service, letting a test adjust the config first — a
// tight limiter, a short heartbeat, a different rule table.
func newEnvWith(t *testing.T, tweak func(*Config)) *env {
	t.Helper()
	ctx := context.Background()

	db, err := Open(ctx, filepath.Join(t.TempDir(), "board.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	clock := newFakeClock()
	store := NewStore(db, clock)
	hub := NewHub(8, 32, quietLogger())

	cfg := Config{
		Store:    store,
		Clock:    clock,
		Hasher:   BcryptHasher{Cost: bcrypt.MinCost}, // production cost is deliberately slow
		Hub:      hub,
		Policy:   NewPolicy(DefaultRules, quietLogger()),
		Sessions: NewSessionStore(clock, time.Hour),
		Limiter:  NewLimiter(100, 100, clock), // generous; the limiter tests bring their own
		Logger:   quietLogger(),

		Heartbeat:   time.Second,
		StreamRetry: 3 * time.Second,
	}
	if tweak != nil {
		tweak(&cfg)
	}
	srv := NewServer(cfg)

	e := &env{
		t: t, srv: srv, store: store, hub: hub, clock: clock,
		handler: srv.Handler(), users: make(map[string]User),
	}
	for _, u := range []struct {
		name string
		role Role
	}{
		{"alice", RoleMember},
		{"bob", RoleMember},
		{"iris", RoleAuditor},
		{"root", RoleAdmin},
	} {
		user, err := srv.Register(ctx, u.name, testPassword, u.role)
		if err != nil {
			t.Fatalf("Register(%s): %v", u.name, err)
		}
		e.users[u.name] = user
	}
	e.pool = &Pool{
		Store:    store,
		Clock:    clock,
		Hub:      hub,
		Handlers: JobHandlers(store, clock),
		Backoff:  Backoff{Base: time.Second, Max: time.Minute, Rand: func(int64) int64 { return 0 }},
		Workers:  1,
		Lease:    30 * time.Second,
		Idle:     time.Millisecond,
		Logger:   quietLogger(),
	}
	return e
}

// subject builds the Subject a fixture user's session would produce, for the
// tests that ask the policy directly.
func (e *env) subject(name string) Subject {
	u := e.users[name]
	return Subject{UserID: u.ID, Role: u.Role, SessionID: "s-" + name}
}

const testAddr = "192.0.2.10:4000"

// request builds a request with a JSON body when there is one.
func (e *env) request(method, path string, body any) *http.Request {
	var r *http.Request
	switch b := body.(type) {
	case nil:
		r = httptest.NewRequest(method, path, nil)
	case string:
		r = httptest.NewRequest(method, path, strings.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			e.t.Fatalf("marshal body: %v", err)
		}
		r = httptest.NewRequest(method, path, bytes.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
	}
	r.RemoteAddr = testAddr
	return r
}

// do serves one request, optionally as a logged-in user.
func (e *env) do(method, path string, body any, c *http.Cookie) *httptest.ResponseRecorder {
	e.t.Helper()
	r := e.request(method, path, body)
	if c != nil {
		r.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, r)
	return rec
}

// doFrom is do from a chosen peer address, for the tests that care what a
// "client" means to the rate limiter.
func (e *env) doFrom(addr, method, path string, body any, c *http.Cookie) *httptest.ResponseRecorder {
	e.t.Helper()
	r := e.request(method, path, body)
	r.RemoteAddr = addr
	if c != nil {
		r.AddCookie(c)
	}
	return e.serve(r)
}

// serve is do for a request the caller built itself.
func (e *env) serve(r *http.Request) *httptest.ResponseRecorder {
	e.t.Helper()
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, r)
	return rec
}

// login authenticates a fixture account and returns its session cookie.
func (e *env) login(username string) *http.Cookie {
	e.t.Helper()
	rec := e.do(http.MethodPost, "/login", credentials{Username: username, Password: testPassword}, nil)
	if rec.Code != http.StatusOK {
		e.t.Fatalf("login(%s): status = %d, want 200: body = %s", username, rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookieName && c.Value != "" {
			return c
		}
	}
	e.t.Fatalf("login(%s): no %s cookie was set", username, SessionCookieName)
	return nil
}

// seedTask writes a task straight to storage, so tests of reading and streaming
// do not depend on the create handler being finished.
func (e *env) seedTask(owner, title string) Task {
	e.t.Helper()
	ctx := context.Background()
	e.seq++
	e.clock.Advance(time.Second) // distinct created_at, so ordering is total
	now := e.clock.Now()
	task := Task{
		ID:        "seed-" + strconv.Itoa(e.seq),
		OwnerID:   e.users[owner].ID,
		Title:     title,
		State:     StateOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
	tx, err := e.store.DB.BeginTx(ctx, nil)
	if err != nil {
		e.t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	if err := e.store.InsertTask(ctx, tx, task); err != nil {
		e.t.Fatalf("seed task: %v", err)
	}
	if err := tx.Commit(); err != nil {
		e.t.Fatalf("commit: %v", err)
	}
	return task
}

// seedTasks writes n tasks for one owner in a single transaction, for the
// paging tests.
func (e *env) seedTasks(owner string, n int) []Task {
	e.t.Helper()
	ctx := context.Background()
	tx, err := e.store.DB.BeginTx(ctx, nil)
	if err != nil {
		e.t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()

	tasks := make([]Task, 0, n)
	for i := 0; i < n; i++ {
		e.seq++
		e.clock.Advance(time.Second)
		now := e.clock.Now()
		task := Task{
			ID:        "seed-" + strconv.Itoa(e.seq),
			OwnerID:   e.users[owner].ID,
			Title:     "task " + strconv.Itoa(e.seq),
			State:     StateOpen,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := e.store.InsertTask(ctx, tx, task); err != nil {
			e.t.Fatalf("seed task: %v", err)
		}
		tasks = append(tasks, task)
	}
	if err := tx.Commit(); err != nil {
		e.t.Fatalf("commit: %v", err)
	}
	return tasks
}

// decodeData pulls the "data" half of the success envelope into dst.
func decodeData(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v\nbody: %s", err, rec.Body.String())
	}
	if len(envelope.Data) == 0 {
		t.Fatalf("response has no data field: %s", rec.Body.String())
	}
	if err := json.Unmarshal(envelope.Data, dst); err != nil {
		t.Fatalf("decode data: %v\nbody: %s", err, rec.Body.String())
	}
}

// errorMessage pulls the "error".message half of the failure envelope.
func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Message string       `json:"message"`
			Fields  []FieldError `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v\nbody: %s", err, rec.Body.String())
	}
	return envelope.Error.Message
}

// watch subscribes to the hub directly, so a test can see exactly what reached
// the fan-out — including what did not.
func (e *env) watch() *Subscriber {
	e.t.Helper()
	sub, err := e.hub.Subscribe()
	if err != nil {
		e.t.Fatalf("Subscribe: %v", err)
	}
	e.t.Cleanup(func() { e.hub.Unsubscribe(sub) })
	return sub
}

// nextEvent takes one published event, or fails.
func nextEvent(t *testing.T, sub *Subscriber) Event {
	t.Helper()
	select {
	case e, ok := <-sub.Events():
		if !ok {
			t.Fatal("the subscription ended, want an event")
		}
		return e
	case <-time.After(hangGuard):
		t.Fatal("no event was published")
		return Event{}
	}
}

// noEvent asserts that nothing has been published. Call it only after
// something that must have completed, so the answer is not a race.
func noEvent(t *testing.T, sub *Subscriber) {
	t.Helper()
	select {
	case e := <-sub.Events():
		t.Fatalf("an event reached subscribers, want none: %+v", e)
	default:
	}
}

func countRows(t *testing.T, e *env, table string) int {
	t.Helper()
	var n int
	if err := e.store.DB.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}
