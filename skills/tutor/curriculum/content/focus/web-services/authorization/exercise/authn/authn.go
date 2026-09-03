// Package authn stands in for the authentication you built last lesson. Its
// only job here is to answer "who is this?" and put the answer in the request
// context; deciding what they may do is package authz's problem.
package authn

import (
	"net/http"
	"strings"

	"tutor.local/authorization/authz"
)

// DemoTokens is NOT how you authenticate anyone: bearer tokens in source, no
// hashing, no expiry, no rotation. Last lesson built the real thing (hashed
// credentials, a session store, an injected clock). It is a fixture so this
// lesson can spend its budget on authorization.
var DemoTokens = map[string]authz.Subject{
	"alice-token": {UserID: "alice", Role: authz.RoleViewer},
	"bob-token":   {UserID: "bob", Role: authz.RoleEditor},
	"dave-token":  {UserID: "dave", Role: authz.RoleEditor},
	"carol-token": {UserID: "carol", Role: authz.RoleAdmin},
}

// Middleware identifies the caller and annotates the request. Note what it
// does not do: reject anybody. An unknown or missing token simply leaves no
// subject in the context, and the authorization layer denies from there. One
// gate, one place to audit — and public routes stay possible without carving
// exceptions into the authentication chain.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if ok {
			if sub, known := DemoTokens[token]; known {
				r = r.WithContext(authz.WithSubject(r.Context(), sub))
			}
		}
		next.ServeHTTP(w, r)
	})
}
