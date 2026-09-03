package harden

import (
	"errors"
	"net/http"
	"time"
)

// CORSPolicy describes which other web origins a browser may let read this
// API's responses. An origin is scheme + host + port: "https://app.example.com"
// and "https://app.example.com:8443" are different origins.
type CORSPolicy struct {
	// AllowedOrigins holds exact origins, or the single entry "*".
	AllowedOrigins []string
	// AllowedMethods are the methods a preflight may ask for.
	AllowedMethods []string
	// AllowedHeaders are the request headers a preflight may ask for.
	AllowedHeaders []string
	// ExposedHeaders are the response headers a page may read. Without this a
	// browser hands the page only the seven safelisted headers, so a
	// Retry-After or an ETag your client depends on may as well not exist.
	ExposedHeaders []string
	// AllowCredentials lets the browser attach cookies and send the response
	// to the page. It is incompatible with a wildcard origin.
	AllowCredentials bool
	// MaxAge is how long a browser may cache a preflight result.
	MaxAge time.Duration
}

// ErrCredentialedWildcard rejects the single most common CORS bug: "*" plus
// credentials would let every site on the internet drive this API as the
// logged-in user. Browsers refuse the combination; refuse it yourself at
// startup, where a human is watching, instead of at request time.
var ErrCredentialedWildcard = errors.New("cors: wildcard origin cannot be combined with credentials")

// NewCORS returns a middleware enforcing p, or an error if p is unsafe.
//
// The contract it must implement:
//
//   - every response carries Vary: Origin, so a shared cache never serves one
//     origin's response to another;
//   - a request with no Origin header is not a cross-origin browser request:
//     pass it through untouched;
//   - a preflight is OPTIONS carrying Access-Control-Request-Method. Answer it
//     with 204 and never call next;
//   - an allowed origin gets Access-Control-Allow-Origin echoed back exactly
//     (or "*" when the policy is wildcard and credential-free), plus
//     Access-Control-Allow-Credentials when the policy says so;
//   - an allowed origin's ordinary (non-preflight) response also gets
//     Access-Control-Expose-Headers when the policy names any;
//   - a preflight for an allowed origin and an allowed method also gets
//     Access-Control-Allow-Methods, Access-Control-Allow-Headers and
//     Access-Control-Max-Age;
//   - anything else gets no Access-Control-* headers at all. Withholding them
//     is the denial: there is no "CORS reject" status code.
func NewCORS(p CORSPolicy) (Middleware, error) {
	// TODO: validate p, precompute the origin and method sets once here
	// rather than per request, and return the middleware described above.
	return func(next http.Handler) http.Handler { return next }, nil
}
