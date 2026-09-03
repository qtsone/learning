package auth

import (
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestRegister(t *testing.T) {
	e := newEnv(t)

	user := e.register("ada", testPassword)
	if user.ID == "" || user.Username != "ada" || user.Role != RoleUser {
		t.Fatalf("Register() = %+v, want an id, username ada and role %q", user, RoleUser)
	}
	if user.PasswordHash == "" || user.PasswordHash == testPassword {
		t.Fatalf("stored PasswordHash = %q, want a hash of the password", user.PasswordHash)
	}

	cases := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{"username too short", "ab", testPassword, ErrInvalidUsername},
		{"username is only spaces", "     ", testPassword, ErrInvalidUsername},
		{"password too short", "bob", "short", ErrPasswordTooShort},
		{"password over the bcrypt limit", "bob", strings.Repeat("x", MaxPasswordBytes+1), ErrPasswordTooLong},
		{"username already taken (case-insensitively)", "ADA", testPassword, ErrUserExists},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := e.svc.Register(c.username, c.password); !errors.Is(err, c.wantErr) {
				t.Errorf("Register(%q, …) = error %v, want %v", c.username, err, c.wantErr)
			}
		})
	}
}

func TestLoginIssuesAHardenedSessionCookie(t *testing.T) {
	e := newEnv(t)
	user := e.register("ada", testPassword)

	res := e.login("ada", testPassword)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /login = %d, want %d", res.StatusCode, http.StatusOK)
	}
	body := readBody(t, res)
	if got := decodeUser(t, body); got.ID != user.ID || got.Role != RoleUser {
		t.Errorf("login body = %+v, want the logged-in user %+v", got, user)
	}
	if bytes.Contains(body, []byte("$2")) || bytes.Contains(body, []byte(testPassword)) {
		t.Errorf("login body %s leaks credential material", body)
	}

	c := sessionCookie(t, res)
	if c.Value == "" {
		t.Errorf("session cookie has an empty value")
	}
	if !c.HttpOnly {
		t.Errorf("session cookie is not HttpOnly: JavaScript can read it")
	}
	if !c.Secure {
		t.Errorf("session cookie is not Secure although CookieSecure is set: it would travel over plain HTTP")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("session cookie SameSite = %v, want %v (http.SameSiteLaxMode)", c.SameSite, http.SameSiteLaxMode)
	}
	if c.Path != "/" {
		t.Errorf("session cookie Path = %q, want \"/\"", c.Path)
	}
	if c.MaxAge != int(testTTL.Seconds()) {
		t.Errorf("session cookie MaxAge = %d, want %d (the session TTL in seconds)", c.MaxAge, int(testTTL.Seconds()))
	}

	sess, ok := e.svc.Sessions().Lookup(c.Value)
	if !ok {
		t.Fatalf("the cookie value names no session in the store")
	}
	if sess.UserID != user.ID {
		t.Errorf("session belongs to %q, want %q", sess.UserID, user.ID)
	}
	if want := testStart.Add(testTTL); !sess.ExpiresAt.Equal(want) {
		t.Errorf("session expires at %v, want %v (the injected clock plus the TTL)", sess.ExpiresAt, want)
	}
}

func TestLoginFailuresRevealNothing(t *testing.T) {
	e := newEnv(t)
	e.register("ada", testPassword)

	before := e.hasher.count()
	wrong := e.login("ada", "not the right password")
	wrongBody := readBody(t, wrong)
	afterWrong := e.hasher.count()

	unknown := e.login("nobody", "not the right password")
	unknownBody := readBody(t, unknown)
	afterUnknown := e.hasher.count()

	for _, res := range []*http.Response{wrong, unknown} {
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("failed login = %d, want %d", res.StatusCode, http.StatusUnauthorized)
		}
		if findCookie(res, SessionCookieName) != nil {
			t.Errorf("a failed login handed out a session cookie")
		}
	}
	if !bytes.Equal(wrongBody, unknownBody) {
		t.Errorf("wrong password answered %s but unknown user answered %s; the two must be indistinguishable", wrongBody, unknownBody)
	}
	if msg := errorMessage(t, wrongBody); msg != "invalid credentials" {
		t.Errorf("failed login message = %q, want %q", msg, "invalid credentials")
	}
	if got := afterWrong - before; got != 1 {
		t.Errorf("wrong password cost %d password verification(s), want 1", got)
	}
	if got := afterUnknown - afterWrong; got != 1 {
		t.Errorf("unknown username cost %d password verification(s), want 1: verify against the dummy hash so both paths do the same work", got)
	}
}

func TestLoginRejectsMalformedJSON(t *testing.T) {
	e := newEnv(t)
	res := e.post("/login", "{not json")
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /login with a broken body = %d, want %d", res.StatusCode, http.StatusBadRequest)
	}
	if msg := errorMessage(t, readBody(t, res)); msg != "invalid JSON" {
		t.Errorf("message = %q, want %q", msg, "invalid JSON")
	}
}

func TestLoginNeverAdoptsAClientSuppliedSessionID(t *testing.T) {
	e := newEnv(t)
	e.register("ada", testPassword)

	planted := &http.Cookie{Name: SessionCookieName, Value: "id-chosen-by-an-attacker"}
	res := e.login("ada", testPassword, planted)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /login = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := sessionCookie(t, res).Value; got == planted.Value {
		t.Errorf("login kept the client-supplied id %q: that is session fixation", got)
	}
	if n := e.svc.Sessions().Len(); n != 1 {
		t.Errorf("store holds %d session(s) after one login, want 1", n)
	}
}

func TestLoginDestroysThePreviousSession(t *testing.T) {
	e := newEnv(t)
	e.register("ada", testPassword)

	first := sessionCookie(t, e.login("ada", testPassword))
	second := sessionCookie(t, e.login("ada", testPassword, first))

	if first.Value == second.Value {
		t.Fatalf("logging in twice reused the session id")
	}
	if _, ok := e.svc.Sessions().Lookup(first.Value); ok {
		t.Errorf("the previous session is still live after a new login")
	}
	if res := e.get("/me", first); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /me with the old cookie = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
	if n := e.svc.Sessions().Len(); n != 1 {
		t.Errorf("store holds %d session(s) after two logins by one user, want 1", n)
	}
}

func TestAuthenticateMiddleware(t *testing.T) {
	e := newEnv(t)
	user := e.register("ada", testPassword)
	valid := sessionCookie(t, e.login("ada", testPassword))

	t.Run("no cookie", func(t *testing.T) {
		res := e.get("/me")
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("GET /me without a cookie = %d, want %d", res.StatusCode, http.StatusUnauthorized)
		}
		if msg := errorMessage(t, readBody(t, res)); msg != "authentication required" {
			t.Errorf("message = %q, want %q", msg, "authentication required")
		}
	})

	t.Run("unknown session id", func(t *testing.T) {
		res := e.get("/me", &http.Cookie{Name: SessionCookieName, Value: "no-such-session"})
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("GET /me with an unknown id = %d, want %d", res.StatusCode, http.StatusUnauthorized)
		}
		cleared := findCookie(res, SessionCookieName)
		if cleared == nil || cleared.MaxAge >= 0 {
			t.Errorf("a dead session cookie was not cleared (got %+v), want MaxAge below zero", cleared)
		}
	})

	t.Run("valid session", func(t *testing.T) {
		res := e.get("/me", valid)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET /me with a live session = %d, want %d", res.StatusCode, http.StatusOK)
		}
		body := readBody(t, res)
		if got := decodeUser(t, body); got.ID != user.ID || got.Username != "ada" {
			t.Errorf("GET /me = %+v, want %+v", got, user)
		}
		if bytes.Contains(body, []byte("$2")) || bytes.Contains(bytes.ToLower(body), []byte("passwordhash")) {
			t.Errorf("GET /me body %s exposes the password hash", body)
		}
	})

	t.Run("expired session", func(t *testing.T) {
		e.clock.Advance(testTTL)
		if res := e.get("/me", valid); res.StatusCode != http.StatusUnauthorized {
			t.Errorf("GET /me after the TTL elapsed = %d, want %d", res.StatusCode, http.StatusUnauthorized)
		}
		if n := e.svc.Sessions().Len(); n != 0 {
			t.Errorf("store holds %d session(s) after the expired lookup, want 0", n)
		}
	})
}

func TestLogout(t *testing.T) {
	e := newEnv(t)
	e.register("ada", testPassword)
	c := sessionCookie(t, e.login("ada", testPassword))

	res := e.post("/logout", "", c)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /logout = %d, want %d", res.StatusCode, http.StatusNoContent)
	}
	if body := readBody(t, res); len(body) != 0 {
		t.Errorf("POST /logout body = %q, want empty (204 promises no body)", body)
	}
	cleared := findCookie(res, SessionCookieName)
	if cleared == nil || cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Errorf("logout cookie = %+v, want an empty value with MaxAge below zero", cleared)
	}
	if n := e.svc.Sessions().Len(); n != 0 {
		t.Errorf("store holds %d session(s) after logout, want 0: the server must forget it too", n)
	}
	if res := e.get("/me", c); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /me after logout = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
	if res := e.post("/logout", ""); res.StatusCode != http.StatusNoContent {
		t.Errorf("POST /logout without a session = %d, want %d (logout is idempotent)", res.StatusCode, http.StatusNoContent)
	}
}

func TestPromoteRotatesTheSessionID(t *testing.T) {
	e := newEnv(t)
	e.register("ada", testPassword)
	before := sessionCookie(t, e.login("ada", testPassword))

	res := e.post("/me/promote", "", before)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /me/promote = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := decodeUser(t, readBody(t, res)); got.Role != RoleAdmin {
		t.Errorf("promoted user role = %q, want %q", got.Role, RoleAdmin)
	}

	after := sessionCookie(t, res)
	if after.Value == before.Value {
		t.Fatalf("the session id survived a privilege change; it must be rotated")
	}
	if n := e.svc.Sessions().Len(); n != 1 {
		t.Errorf("store holds %d session(s) after rotation, want 1", n)
	}
	if res := e.get("/me", before); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /me with the pre-promotion cookie = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
	res = e.get("/me", after)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /me with the rotated cookie = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := decodeUser(t, readBody(t, res)); got.Role != RoleAdmin {
		t.Errorf("GET /me role = %q, want %q", got.Role, RoleAdmin)
	}
}
