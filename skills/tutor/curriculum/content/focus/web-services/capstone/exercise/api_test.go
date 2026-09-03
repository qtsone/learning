package board

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAnonymousAndUnknownSessionsAre401(t *testing.T) {
	e := newEnv(t)

	rec := e.do(http.MethodGet, "/tasks", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /tasks with no cookie = %d, want 401", rec.Code)
	}
	if msg := errorMessage(t, rec); msg != "unauthenticated" {
		t.Errorf("message = %q, want %q", msg, "unauthenticated")
	}

	rec = e.do(http.MethodGet, "/tasks", nil, &http.Cookie{Name: SessionCookieName, Value: "not-a-session"})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /tasks with an unknown session = %d, want 401", rec.Code)
	}

	if rec := e.do(http.MethodGet, "/tasks", nil, e.login("alice")); rec.Code != http.StatusOK {
		t.Errorf("GET /tasks as a member = %d, want 200: body = %s", rec.Code, rec.Body.String())
	}
}

// Authentication rejects nobody: it establishes an identity or leaves the
// anonymous zero value, and every refusal comes out of the authorization layer.
// What it does do for a dead session is tell the browser to drop the key to a
// lock that no longer exists.
func TestADeadSessionCookieIsCleared(t *testing.T) {
	e := newEnv(t)

	rec := e.do(http.MethodGet, "/tasks", nil, &http.Cookie{Name: SessionCookieName, Value: "expired-long-ago"})

	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookieName && c.MaxAge < 0 {
			return
		}
	}
	t.Errorf("no cleared %s cookie on the 401: a cookie naming a session that is gone should be cleared", SessionCookieName)
}

func TestTheRouteGateRefusesBeforeTheHandlerRuns(t *testing.T) {
	e := newEnv(t)

	rec := e.do(http.MethodPost, "/tasks", CreateTaskRequest{Title: "audit finding"}, e.login("iris"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("auditor POST /tasks = %d, want 403: body = %s", rec.Code, rec.Body.String())
	}
	if msg := errorMessage(t, rec); msg != "forbidden" {
		t.Errorf("message = %q, want %q", msg, "forbidden")
	}
	if n := countRows(t, e, "tasks"); n != 0 {
		t.Errorf("tasks = %d, want 0: a denied request must not reach the handler", n)
	}

	if rec := e.do(http.MethodPost, "/tasks", CreateTaskRequest{Title: "ship it"}, e.login("alice")); rec.Code != http.StatusCreated {
		t.Errorf("member POST /tasks = %d, want 201: body = %s", rec.Code, rec.Body.String())
	}
	if rec := e.do(http.MethodGet, "/tasks", nil, e.login("iris")); rec.Code != http.StatusOK {
		t.Errorf("auditor GET /tasks = %d, want 200", rec.Code)
	}
}

// The delete route exists and really deletes. Nothing but the missing rule
// stops it — which is the whole value of deny-by-default: the case nobody
// thought about fails closed.
func TestDeleteIsDeniedForEveryRoleIncludingAdmin(t *testing.T) {
	e := newEnv(t)
	task := e.seedTask("alice", "do not delete me")

	for _, name := range []string{"alice", "iris", "root"} {
		rec := e.do(http.MethodDelete, "/tasks/"+task.ID, nil, e.login(name))
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s DELETE /tasks/{id} = %d, want 403: no rule grants task:delete", name, rec.Code)
		}
		if _, err := e.store.TaskByID(context.Background(), task.ID); err != nil {
			t.Fatalf("the task was deleted by %s: %v", name, err)
		}
	}
}

func TestOwnershipIsEnforcedOnTheLoadedObject(t *testing.T) {
	e := newEnv(t)
	task := e.seedTask("alice", "alice's task")
	body := UpdateTaskRequest{State: StateDone}

	cases := []struct {
		user       string
		method     string
		wantStatus int
		why        string
	}{
		{"bob", http.MethodGet, http.StatusForbidden, "a valid session with a permitted role is not permission to read someone else's row"},
		{"bob", http.MethodPatch, http.StatusForbidden, "changing the id in the URL must not change whose data you reach"},
		{"iris", http.MethodGet, http.StatusOK, "auditors are exempt from ownership on reads"},
		{"iris", http.MethodPatch, http.StatusForbidden, "auditors have no update rule, exemption or not"},
		{"alice", http.MethodGet, http.StatusOK, "the owner may read it"},
		{"root", http.MethodPatch, http.StatusOK, "admins are exempt from ownership"},
	}
	for _, tc := range cases {
		t.Run(tc.user+" "+tc.method, func(t *testing.T) {
			var rec *httptest.ResponseRecorder
			if tc.method == http.MethodGet {
				rec = e.do(http.MethodGet, "/tasks/"+task.ID, nil, e.login(tc.user))
			} else {
				rec = e.do(http.MethodPatch, "/tasks/"+task.ID, body, e.login(tc.user))
			}
			if rec.Code != tc.wantStatus {
				t.Fatalf("%s %s = %d, want %d (%s): body = %s",
					tc.user, tc.method, rec.Code, tc.wantStatus, tc.why, rec.Body.String())
			}
			if rec.Code != http.StatusForbidden {
				return
			}
			stored, err := e.store.TaskByID(context.Background(), task.ID)
			if err != nil {
				t.Fatalf("load task: %v", err)
			}
			if stored.State != StateOpen {
				t.Errorf("state = %q after a denied request, want %q: a denial must leave storage untouched",
					stored.State, StateOpen)
			}
			if reason := rec.Body.String(); strings.Contains(reason, "owner") || strings.Contains(reason, "role") {
				t.Errorf("the denial body names the reason: %s\nthe reason belongs in the log; in a response it confirms "+
					"that the object exists and belongs to somebody else", reason)
			}
		})
	}

	if rec := e.do(http.MethodGet, "/tasks/nope", nil, e.login("root")); rec.Code != http.StatusNotFound {
		t.Errorf("GET a task that does not exist = %d, want 404", rec.Code)
	}
}

func TestListReturnsOnlyWhatTheCallerMayRead(t *testing.T) {
	e := newEnv(t)
	e.seedTask("alice", "alice one")
	e.seedTask("alice", "alice two")
	e.seedTask("bob", "bob one")

	cases := []struct {
		user  string
		count int
	}{{"alice", 2}, {"bob", 1}, {"iris", 3}, {"root", 3}}
	for _, tc := range cases {
		rec := e.do(http.MethodGet, "/tasks", nil, e.login(tc.user))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s GET /tasks = %d: body = %s", tc.user, rec.Code, rec.Body.String())
		}
		var page TaskPage
		decodeData(t, rec, &page)
		if len(page.Items) != tc.count {
			t.Errorf("%s sees %d task(s), want %d: the listing must be restricted to what the policy allows, "+
				"and the restriction belongs in the query", tc.user, len(page.Items), tc.count)
		}
		for _, item := range page.Items {
			if tc.user == "alice" && item.OwnerID != e.users["alice"].ID {
				t.Errorf("alice was shown a task owned by %s", item.OwnerID)
			}
		}
	}
}

func TestListPagesWithAKeysetCursor(t *testing.T) {
	e := newEnv(t)
	e.seedTasks("alice", 5)
	cookie := e.login("alice")

	seen := map[string]bool{}
	path := "/tasks?limit=2"
	for page := 1; ; page++ {
		rec := e.do(http.MethodGet, path, nil, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("page %d = %d: body = %s", page, rec.Code, rec.Body.String())
		}
		var got TaskPage
		decodeData(t, rec, &got)
		if got.Items == nil {
			t.Fatal("items is null, want an empty array at worst")
		}
		if len(got.Items) > 2 {
			t.Fatalf("page %d has %d items, want at most the requested 2", page, len(got.Items))
		}
		for _, item := range got.Items {
			if seen[item.ID] {
				t.Fatalf("task %s appeared on two pages", item.ID)
			}
			seen[item.ID] = true
		}
		if got.NextCursor == "" {
			break
		}
		if page > 5 {
			t.Fatal("still paging after 5 pages: next_cursor must be empty once the last page is served")
		}
		path = "/tasks?limit=2&cursor=" + got.NextCursor
	}
	if len(seen) != 5 {
		t.Errorf("paged over %d tasks, want 5: no row may be skipped", len(seen))
	}
}

func TestListClampsAndValidatesItsParameters(t *testing.T) {
	e := newEnv(t)
	e.seedTasks("alice", MaxPageLimit+1)
	cookie := e.login("alice")

	rec := e.do(http.MethodGet, "/tasks?limit=1000", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: body = %s", rec.Code, rec.Body.String())
	}
	var page TaskPage
	decodeData(t, rec, &page)
	if len(page.Items) > MaxPageLimit {
		t.Errorf("limit=1000 returned %d items, want at most MaxPageLimit (%d): a caller must not decide how much "+
			"memory you spend", len(page.Items), MaxPageLimit)
	}

	noColon := base64.RawURLEncoding.EncodeToString([]byte("no-separator-here"))
	for _, query := range []string{"?limit=nope", "?limit=0", "?cursor=not-base64!", "?cursor=" + noColon} {
		if rec := e.do(http.MethodGet, "/tasks"+query, nil, cookie); rec.Code != http.StatusBadRequest {
			t.Errorf("GET /tasks%s = %d, want 400: an unreadable parameter is the client's mistake", query, rec.Code)
		}
	}
}

func TestListIsCacheableWithoutLeakingBetweenUsers(t *testing.T) {
	e := newEnv(t)
	e.seedTask("alice", "alice one")
	e.seedTask("bob", "bob one")
	alice, iris := e.login("alice"), e.login("iris")

	rec := e.do(http.MethodGet, "/tasks", nil, alice)
	tag := rec.Header().Get("ETag")
	if tag == "" {
		t.Fatal("no ETag on the listing: a client that already holds these bytes should be able to say so")
	}
	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl == "" || strings.Contains(cacheControl, "public") {
		t.Errorf("Cache-Control = %q: this body depends on who asked, so no shared cache may store it", cacheControl)
	}
	if vary := rec.Header().Get("Vary"); !strings.Contains(vary, "Cookie") {
		t.Errorf("Vary = %q, want it to name Cookie: the response depends on the session, and a cache that does not "+
			"know that will hand one user another user's tasks", vary)
	}

	r := e.request(http.MethodGet, "/tasks", nil)
	r.AddCookie(alice)
	r.Header.Set("If-None-Match", tag)
	rec = e.serve(r)
	if rec.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match with the current tag = %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("304 carried a %d-byte body, want none", rec.Body.Len())
	}
	if got := rec.Header().Get("ETag"); got != tag {
		t.Errorf("ETag on the 304 = %q, want %q", got, tag)
	}

	if irisTag := e.do(http.MethodGet, "/tasks", nil, iris).Header().Get("ETag"); irisTag == tag {
		t.Error("the auditor's listing has the same ETag as the member's, but they must not contain the same rows")
	}

	if rec := e.do(http.MethodPost, "/tasks", CreateTaskRequest{Title: "new"}, alice); rec.Code != http.StatusCreated {
		t.Fatalf("POST /tasks = %d: body = %s", rec.Code, rec.Body.String())
	}
	r = e.request(http.MethodGet, "/tasks", nil)
	r.AddCookie(alice)
	r.Header.Set("If-None-Match", tag)
	if rec := e.serve(r); rec.Code != http.StatusOK {
		t.Errorf("If-None-Match with a stale tag = %d, want 200: the listing changed", rec.Code)
	}
}

func TestCreateValidatesTheBody(t *testing.T) {
	e := newEnv(t)
	cookie := e.login("alice")

	cases := []struct {
		name string
		body any
		want int
	}{
		{"unknown field", `{"title":"x","owner_id":"u-bob"}`, http.StatusBadRequest},
		{"blank title", CreateTaskRequest{Title: "   "}, http.StatusUnprocessableEntity},
		{"title too long", CreateTaskRequest{Title: strings.Repeat("a", MaxTitleRunes+1)}, http.StatusUnprocessableEntity},
		{"two JSON values", `{"title":"one"}{"title":"two"}`, http.StatusBadRequest},
		{"oversized body", `{"title":"` + strings.Repeat("a", int(DefaultMaxBodyBytes)+100) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := e.do(http.MethodPost, "/tasks", tc.body, cookie); rec.Code != tc.want {
				t.Errorf("status = %d, want %d: body = %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}

	r := e.request(http.MethodPost, "/tasks", `{"title":"x"}`)
	r.Header.Set("Content-Type", "text/plain")
	r.AddCookie(cookie)
	if rec := e.serve(r); rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("text/plain body = %d, want 415", rec.Code)
	}
	if n := countRows(t, e, "tasks"); n != 0 {
		t.Errorf("tasks = %d, want 0: nothing above was a valid create", n)
	}
}

func TestCreateTakesOwnershipFromTheSession(t *testing.T) {
	e := newEnv(t)

	rec := e.do(http.MethodPost, "/tasks", CreateTaskRequest{Title: "ship it"}, e.login("alice"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: body = %s", rec.Code, rec.Body.String())
	}
	var task Task
	decodeData(t, rec, &task)
	if task.OwnerID != e.users["alice"].ID {
		t.Errorf("owner = %q, want %q: ownership comes from the session, never from the body",
			task.OwnerID, e.users["alice"].ID)
	}
	if task.State != StateOpen || task.CreatedAt.IsZero() {
		t.Errorf("task = %+v, want a new open task with a timestamp from the injected clock", task)
	}
}

func TestRateLimitKeysOnTheAccountThenTheAddress(t *testing.T) {
	e := newEnvWith(t, func(c *Config) { c.Limiter = NewLimiter(1, 2, c.Clock) })

	alice, bob := e.login("alice"), e.login("bob") // two tokens from the address bucket

	for i := 1; i <= 2; i++ {
		if rec := e.do(http.MethodGet, "/tasks", nil, alice); rec.Code != http.StatusOK {
			t.Fatalf("alice request %d = %d, want 200: a fresh key starts with a full burst", i, rec.Code)
		}
	}
	rec := e.do(http.MethodGet, "/tasks", nil, alice)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("alice's third request = %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got == "" || got == "0" {
		t.Errorf("Retry-After = %q, want whole seconds, at least 1", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options on the 429 = %q, want nosniff: the security headers belong outermost, "+
			"so every response carries them — including the ones no handler produced", got)
	}

	if rec := e.do(http.MethodGet, "/tasks", nil, bob); rec.Code != http.StatusOK {
		t.Errorf("bob = %d, want 200: two accounts must not share a bucket", rec.Code)
	}

	// The address bucket was spent by the two logins, and an anonymous caller
	// is exactly who that key is for.
	if rec := e.do(http.MethodPost, "/login", credentials{Username: "root", Password: testPassword}, nil); rec.Code != http.StatusTooManyRequests {
		t.Errorf("a third login from the same address = %d, want 429: anonymous callers are limited by address", rec.Code)
	}
	if rec := e.doFrom("198.51.100.7:5000", http.MethodPost, "/login", credentials{Username: "root", Password: testPassword}, nil); rec.Code != http.StatusOK {
		t.Errorf("login from another address = %d, want 200: addresses are independent keys", rec.Code)
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

// A caller over its budget must cost a map lookup, not a body read and a JSON
// decode. If parsing runs first, an attacker gets the expensive work anyway.
func TestRateLimitRefusesBeforeTouchingTheBody(t *testing.T) {
	e := newEnvWith(t, func(c *Config) { c.Limiter = NewLimiter(1, 1, c.Clock) })
	cookie := e.login("alice")

	if rec := e.do(http.MethodPost, "/tasks", CreateTaskRequest{Title: "first"}, cookie); rec.Code != http.StatusCreated {
		t.Fatalf("first request = %d, want 201: body = %s", rec.Code, rec.Body.String())
	}

	body := &countingReader{data: strings.NewReader(`{"title":"second"}`)}
	r := httptest.NewRequest(http.MethodPost, "/tasks", body)
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = testAddr
	r.AddCookie(cookie)

	rec := e.serve(r)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request = %d, want 429", rec.Code)
	}
	if n := body.count(); n != 0 {
		t.Errorf("the body was read %d time(s) for a refused request, want 0: rate limiting goes outside parsing", n)
	}
}

// The object check goes before the decode, not after it. A caller who may not
// touch this row should cost a lookup and a map read — not that plus a 64 KiB
// read and a strict decode, which is a denied request paying for the expensive
// half anyway.
func TestADeniedUpdateNeverReadsTheBody(t *testing.T) {
	e := newEnv(t)
	task := e.seedTask("alice", "alice's task")

	body := &countingReader{data: strings.NewReader(`{"state":"done"}`)}
	r := httptest.NewRequest(http.MethodPatch, "/tasks/"+task.ID, body)
	r.Header.Set("Content-Type", "application/json")
	r.RemoteAddr = testAddr
	r.AddCookie(e.login("bob"))

	rec := e.serve(r)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("bob PATCH alice's task = %d, want 403: body = %s", rec.Code, rec.Body.String())
	}
	if n := body.count(); n != 0 {
		t.Errorf("the body was read %d time(s) for a denied request, want 0: load, enforce, then decode", n)
	}
}

// Both of the service's in-memory maps are keyed on something a caller
// controls, so both need something that removes entries. Nothing else does.
func TestTheJanitorEvictsIdleBucketsAndDeadSessions(t *testing.T) {
	e := newEnvWith(t, func(c *Config) {
		c.Sessions = NewSessionStore(c.Clock, 30*time.Minute)
		c.Limiter = NewLimiter(1, 2, c.Clock)
	})
	e.login("alice")
	if n := e.srv.Sessions().Count(); n != 1 {
		t.Fatalf("sessions = %d after one login, want 1", n)
	}

	e.clock.Advance(time.Hour)
	buckets, sessions := e.srv.Sweep()
	if buckets == 0 {
		t.Error("Sweep evicted no rate-limit buckets: the map grows once per distinct key seen, and an anonymous " +
			"caller's key is their address")
	}
	if sessions != 1 {
		t.Errorf("Sweep evicted %d session(s), want 1: expiry on read stops a dead session working but never removes it", sessions)
	}
	if n := e.srv.Sessions().Count(); n != 0 {
		t.Errorf("sessions = %d after the sweep, want 0", n)
	}

	// And the loop that calls it takes its ticks from the injected Clock and
	// stops when its context does, so shutdown does not wait on it.
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); e.srv.StartJanitor(ctx, time.Minute) }()
	e.clock.tick(t)
	cancel()
	select {
	case <-done:
	case <-time.After(hangGuard):
		t.Fatal("StartJanitor did not return when its context was cancelled")
	}
}

func TestSecurityHeadersReachEveryResponse(t *testing.T) {
	e := newEnv(t)

	for _, tc := range []struct {
		name string
		rec  *httptest.ResponseRecorder
	}{
		{"401", e.do(http.MethodGet, "/tasks", nil, nil)},
		{"200", e.do(http.MethodGet, "/tasks", nil, e.login("alice"))},
	} {
		if got := tc.rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q, want nosniff", tc.name, got)
		}
		if got := tc.rec.Header().Get("Content-Security-Policy"); got == "" {
			t.Errorf("%s: no Content-Security-Policy", tc.name)
		}
		if got := tc.rec.Header().Get("X-XSS-Protection"); got != "" {
			t.Errorf("%s: X-XSS-Protection = %q, want it unset: it is dead and its filter had its own holes", tc.name, got)
		}
	}
}

func TestRoleChangeRevokesTheTargetsSessions(t *testing.T) {
	e := newEnv(t)
	e.seedTask("bob", "bob's task")
	alice, root := e.login("alice"), e.login("root")

	if rec := e.do(http.MethodGet, "/tasks", nil, alice); rec.Code != http.StatusOK {
		t.Fatalf("alice before the change = %d, want 200", rec.Code)
	}

	rec := e.do(http.MethodPost, "/users/"+e.users["alice"].ID+"/role", SetRoleRequest{Role: RoleAdmin}, root)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin POST /users/{id}/role = %d, want 200: body = %s", rec.Code, rec.Body.String())
	}
	var updated User
	decodeData(t, rec, &updated)
	if updated.Role != RoleAdmin {
		t.Errorf("role = %q, want %q", updated.Role, RoleAdmin)
	}

	if rec := e.do(http.MethodGet, "/tasks", nil, alice); rec.Code != http.StatusUnauthorized {
		t.Errorf("alice's pre-change cookie = %d, want 401: a session minted under the old privilege must not survive "+
			"the change", rec.Code)
	}
	if rec := e.do(http.MethodGet, "/tasks", nil, root); rec.Code != http.StatusOK {
		t.Errorf("the actor's own session = %d, want 200: only the target's sessions are affected", rec.Code)
	}

	var page TaskPage
	decodeData(t, e.do(http.MethodGet, "/tasks", nil, e.login("alice")), &page)
	if len(page.Items) != 1 {
		t.Errorf("after logging in again alice sees %d task(s), want 1: her new role reads everything", len(page.Items))
	}
}

func TestRoleChangeRequiresAnAdmin(t *testing.T) {
	e := newEnv(t)

	rec := e.do(http.MethodPost, "/users/"+e.users["bob"].ID+"/role", SetRoleRequest{Role: RoleAdmin}, e.login("alice"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member POST /users/{id}/role = %d, want 403", rec.Code)
	}
	user, err := e.store.UserByID(context.Background(), e.users["bob"].ID)
	if err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Role != RoleMember {
		t.Errorf("bob is now %q: a denied request must change nothing", user.Role)
	}
}

func TestSessionsExpireAgainstTheInjectedClock(t *testing.T) {
	e := newEnvWith(t, func(c *Config) { c.Sessions = NewSessionStore(c.Clock, 30*time.Minute) })
	cookie := e.login("alice")

	if rec := e.do(http.MethodGet, "/tasks", nil, cookie); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	e.clock.Advance(31 * time.Minute)
	if rec := e.do(http.MethodGet, "/tasks", nil, cookie); rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d after the session's lifetime, want 401: expiry is enforced server-side, on read", rec.Code)
	}
}
