package api_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"tutor.local/authorization/api"
	"tutor.local/authorization/authz"
	"tutor.local/authorization/doc"
	"tutor.local/authorization/memstore"
)

// Tokens from authn.DemoTokens: alice is a viewer, bob and dave are editors,
// carol is an admin.
const (
	aliceToken = "alice-token"
	bobToken   = "bob-token"
	daveToken  = "dave-token"
	carolToken = "carol-token"
)

func seed() []doc.Document {
	return []doc.Document{
		{ID: "alice-doc", OwnerID: "alice", Title: "Alice's notes"},
		{ID: "bob-doc", OwnerID: "bob", Title: "Bob's draft"},
		{ID: "dave-doc", OwnerID: "dave", Title: "Dave's draft"},
		{ID: "carol-doc", OwnerID: "carol", Title: "Carol's memo"},
	}
}

func newServer(t *testing.T) (http.Handler, *memstore.Store) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return newServerWithLogger(t, log)
}

func newServerWithLogger(t *testing.T, log *slog.Logger) (http.Handler, *memstore.Store) {
	t.Helper()
	store := memstore.New(seed()...)
	policy := authz.NewPolicy(authz.DefaultRules, log)
	return api.New(store, policy).Handler(), store
}

func do(t *testing.T, h http.Handler, token, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, r)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var env struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("body is not a {\"data\": ...} envelope: %v\nbody: %s", err, rec.Body.String())
	}
	return env.Data
}

func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("body is not an {\"error\": {...}} envelope: %v\nbody: %s", err, rec.Body.String())
	}
	return env.Error.Message
}

// every route, as (method, target) pairs, for the sweeps that must hold
// everywhere.
var allRoutes = []struct{ method, target string }{
	{http.MethodGet, "/docs"},
	{http.MethodPost, "/docs"},
	{http.MethodGet, "/docs/bob-doc"},
	{http.MethodPut, "/docs/bob-doc"},
	{http.MethodDelete, "/docs/bob-doc"},
	{http.MethodPost, "/docs/bob-doc/archive"},
}

func TestAnonymousRequestsAreRejected(t *testing.T) {
	h, store := newServer(t)
	for _, rt := range allRoutes {
		t.Run(rt.method+" "+rt.target, func(t *testing.T) {
			rec := do(t, h, "", rt.method, rt.target, `{"title":"x"}`)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d (nobody is authenticated)\nbody: %s",
					rec.Code, http.StatusUnauthorized, rec.Body.String())
			}
			if got := errorMessage(t, rec); got != "unauthenticated" {
				t.Errorf("error message = %q, want %q", got, "unauthenticated")
			}
		})
	}
	if all, _ := store.List(); len(all) != len(seed()) {
		t.Errorf("store holds %d documents after the anonymous sweep, want %d — a handler ran",
			len(all), len(seed()))
	}
}

func TestUnknownTokenIsRejected(t *testing.T) {
	h, _ := newServer(t)
	rec := do(t, h, "mallory-token", http.MethodGet, "/docs", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d for an unknown token", rec.Code, http.StatusUnauthorized)
	}
}

func TestRoleGate(t *testing.T) {
	cases := []struct {
		name       string
		token      string
		method     string
		target     string
		body       string
		wantStatus int
	}{
		{"viewer may list", aliceToken, http.MethodGet, "/docs", "", http.StatusOK},
		{"viewer may read own", aliceToken, http.MethodGet, "/docs/alice-doc", "", http.StatusOK},
		{"viewer may not create", aliceToken, http.MethodPost, "/docs", `{"title":"new"}`, http.StatusForbidden},
		{"viewer may not update own", aliceToken, http.MethodPut, "/docs/alice-doc", `{"title":"edited"}`, http.StatusForbidden},
		{"viewer may not delete own", aliceToken, http.MethodDelete, "/docs/alice-doc", "", http.StatusForbidden},
		{"editor may create", bobToken, http.MethodPost, "/docs", `{"title":"new"}`, http.StatusCreated},
		{"editor may read own", bobToken, http.MethodGet, "/docs/bob-doc", "", http.StatusOK},
		{"editor may update own", bobToken, http.MethodPut, "/docs/bob-doc", `{"title":"edited"}`, http.StatusOK},
		{"editor may delete own", bobToken, http.MethodDelete, "/docs/bob-doc", "", http.StatusNoContent},
		{"admin may create", carolToken, http.MethodPost, "/docs", `{"title":"new"}`, http.StatusCreated},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, _ := newServer(t)
			rec := do(t, h, c.token, c.method, c.target, c.body)
			if rec.Code != c.wantStatus {
				t.Fatalf("%s %s as %s: status = %d, want %d\nbody: %s",
					c.method, c.target, c.token, rec.Code, c.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestViewerCreateAttemptChangesNothing: a denial must stop the handler, not
// just rewrite its status code.
func TestViewerCreateAttemptChangesNothing(t *testing.T) {
	h, store := newServer(t)
	do(t, h, aliceToken, http.MethodPost, "/docs", `{"title":"sneaky"}`)
	all, _ := store.List()
	if len(all) != len(seed()) {
		t.Errorf("store holds %d documents, want %d — the create handler ran despite the denial",
			len(all), len(seed()))
	}
}

// TestOwnershipIsEnforcedPerObject is the IDOR test: bob's token is perfectly
// valid and his role permits updates. He still may not touch alice's document.
func TestOwnershipIsEnforcedPerObject(t *testing.T) {
	cases := []struct {
		name   string
		token  string
		method string
		target string
		body   string
	}{
		{"editor reads another user's document", bobToken, http.MethodGet, "/docs/alice-doc", ""},
		{"editor updates another user's document", bobToken, http.MethodPut, "/docs/alice-doc", `{"title":"owned"}`},
		{"editor deletes another user's document", bobToken, http.MethodDelete, "/docs/alice-doc", ""},
		{"editor updates another editor's document", daveToken, http.MethodPut, "/docs/bob-doc", `{"title":"owned"}`},
		{"viewer reads another user's document", aliceToken, http.MethodGet, "/docs/bob-doc", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, store := newServer(t)
			before, _ := store.List()

			rec := do(t, h, c.token, c.method, c.target, c.body)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s %s: status = %d, want %d\nbody: %s",
					c.method, c.target, rec.Code, http.StatusForbidden, rec.Body.String())
			}
			if got := errorMessage(t, rec); got != "forbidden" {
				t.Errorf("error message = %q, want %q", got, "forbidden")
			}
			if body := rec.Body.String(); strings.Contains(body, authz.ReasonNotOwner) {
				t.Errorf("response leaks the decision reason to the client: %s", body)
			}

			after, _ := store.List()
			if len(after) != len(before) {
				t.Fatalf("documents: %d before, %d after a denied request", len(before), len(after))
			}
			for i := range after {
				if after[i] != before[i] {
					t.Errorf("document %s changed during a denied request: %+v -> %+v",
						after[i].ID, before[i], after[i])
				}
			}
		})
	}
}

func TestAdminOverridesOwnership(t *testing.T) {
	h, _ := newServer(t)

	if rec := do(t, h, carolToken, http.MethodGet, "/docs/bob-doc", ""); rec.Code != http.StatusOK {
		t.Errorf("admin GET /docs/bob-doc: status = %d, want %d", rec.Code, http.StatusOK)
	}
	rec := do(t, h, carolToken, http.MethodPut, "/docs/bob-doc", `{"title":"reviewed","body":"ok"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin PUT /docs/bob-doc: status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := decodeData[doc.Document](t, rec); got.Title != "reviewed" {
		t.Errorf("title after admin update = %q, want %q", got.Title, "reviewed")
	}
	if rec := do(t, h, carolToken, http.MethodDelete, "/docs/dave-doc", ""); rec.Code != http.StatusNoContent {
		t.Errorf("admin DELETE /docs/dave-doc: status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

// TestListIsFilteredByPolicy: passing the route gate says you may list, not
// what you may see. A listing that returns rows the caller could not fetch
// one by one is the same leak with extra steps.
func TestListIsFilteredByPolicy(t *testing.T) {
	cases := []struct {
		token string
		want  []string
	}{
		{aliceToken, []string{"alice-doc"}},
		{bobToken, []string{"bob-doc"}},
		{daveToken, []string{"dave-doc"}},
		{carolToken, []string{"alice-doc", "bob-doc", "carol-doc", "dave-doc"}},
	}
	for _, c := range cases {
		t.Run(c.token, func(t *testing.T) {
			h, _ := newServer(t)
			rec := do(t, h, c.token, http.MethodGet, "/docs", "")
			if rec.Code != http.StatusOK {
				t.Fatalf("GET /docs: status = %d, want %d", rec.Code, http.StatusOK)
			}
			docs := decodeData[[]doc.Document](t, rec)
			var ids []string
			for _, d := range docs {
				ids = append(ids, d.ID)
			}
			if strings.Join(ids, ",") != strings.Join(c.want, ",") {
				t.Errorf("GET /docs as %s returned %v, want %v", c.token, ids, c.want)
			}
		})
	}
}

// TestRouteWithoutARuleIsDenied: the archive route exists and its handler
// works. No rule mentions doc:archive, so nobody may reach it — including the
// admin. Deny-by-default is the difference between "we forgot" and "we shipped
// an open endpoint".
func TestRouteWithoutARuleIsDenied(t *testing.T) {
	for _, token := range []string{aliceToken, bobToken, carolToken} {
		t.Run(token, func(t *testing.T) {
			h, store := newServer(t)
			rec := do(t, h, token, http.MethodPost, "/docs/bob-doc/archive", "")
			if rec.Code != http.StatusForbidden {
				t.Fatalf("POST /docs/bob-doc/archive as %s: status = %d, want %d\nbody: %s",
					token, rec.Code, http.StatusForbidden, rec.Body.String())
			}
			d, err := store.Get("bob-doc")
			if err != nil {
				t.Fatalf("store.Get(bob-doc): %v", err)
			}
			if d.Archived {
				t.Error("the document was archived: the handler ran behind an unruled route")
			}
		})
	}
}

func TestUnknownPathIsNotFound(t *testing.T) {
	h, _ := newServer(t)
	rec := do(t, h, carolToken, http.MethodGet, "/internal/metrics", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /internal/metrics: status = %d, want %d (no route, no handler to reach)",
			rec.Code, http.StatusNotFound)
	}
}

func TestCreateTakesOwnerFromTheSubject(t *testing.T) {
	h, _ := newServer(t)
	rec := do(t, h, bobToken, http.MethodPost, "/docs", `{"title":"mine","body":"x","owner_id":"carol"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /docs: status = %d, want %d\nbody: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	created := decodeData[doc.Document](t, rec)
	if created.OwnerID != "bob" {
		t.Errorf("owner_id = %q, want %q — ownership comes from the session, never the body",
			created.OwnerID, "bob")
	}
}

// TestDecisionsAreAudited: a decision nobody can see afterwards is not an
// access-control system, it is a hope. Both halves of enforcement log the same
// shape.
func TestDecisionsAreAudited(t *testing.T) {
	logs := newCapture()
	h, _ := newServerWithLogger(t, slog.New(logs))

	do(t, h, bobToken, http.MethodPut, "/docs/alice-doc", `{"title":"owned"}`)
	deny := logs.find(func(r record) bool {
		return r.msg == "authorization" && r.attrs["reason"] == authz.ReasonNotOwner
	})
	if deny == nil {
		t.Fatalf("no audit record with reason %q; got %v", authz.ReasonNotOwner, logs.summary())
	}
	for key, want := range map[string]any{
		"allow":    false,
		"user":     "bob",
		"role":     string(authz.RoleEditor),
		"action":   string(authz.ActionUpdate),
		"resource": "alice-doc",
		"owner":    "alice",
		"method":   http.MethodPut,
		"path":     "/docs/alice-doc",
	} {
		if got := deny.attrs[key]; got != want {
			t.Errorf("audit record attribute %q = %v, want %v", key, got, want)
		}
	}
	if deny.level != slog.LevelWarn {
		t.Errorf("denial logged at %v, want %v", deny.level, slog.LevelWarn)
	}

	do(t, h, bobToken, http.MethodGet, "/docs/bob-doc", "")
	allow := logs.find(func(r record) bool {
		return r.msg == "authorization" && r.attrs["reason"] == authz.ReasonOwner
	})
	if allow == nil {
		t.Fatalf("allows are not audited: no record with reason %q; got %v",
			authz.ReasonOwner, logs.summary())
	}
	if allow.attrs["allow"] != true {
		t.Errorf("audited allow has allow=%v, want true", allow.attrs["allow"])
	}
}

// --- test-only slog.Handler that keeps records in memory ---

type record struct {
	level slog.Level
	msg   string
	attrs map[string]any
}

type capture struct {
	mu      sync.Mutex
	records []record
}

func newCapture() *capture { return &capture{} }

func (c *capture) Enabled(context.Context, slog.Level) bool { return true }

func (c *capture) Handle(_ context.Context, r slog.Record) error {
	rec := record{level: r.Level, msg: r.Message, attrs: map[string]any{}}
	r.Attrs(func(a slog.Attr) bool {
		rec.attrs[a.Key] = a.Value.Any()
		return true
	})
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, rec)
	return nil
}

func (c *capture) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *capture) WithGroup(string) slog.Handler      { return c }

func (c *capture) find(pred func(record) bool) *record {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.records {
		if pred(c.records[i]) {
			found := c.records[i]
			return &found
		}
	}
	return nil
}

func (c *capture) summary() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.records))
	for _, r := range c.records {
		out = append(out, r.msg+":"+toString(r.attrs["reason"]))
	}
	return out
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}
