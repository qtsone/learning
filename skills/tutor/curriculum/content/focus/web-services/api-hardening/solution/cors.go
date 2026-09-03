package harden

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
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
func NewCORS(p CORSPolicy) (Middleware, error) {
	// Everything derived from the policy is computed once, here, because the
	// policy cannot change while the process runs.
	wildcard := false
	origins := make(map[string]bool, len(p.AllowedOrigins))
	for _, o := range p.AllowedOrigins {
		if o == "*" {
			wildcard = true
			continue
		}
		origins[o] = true
	}
	if wildcard && p.AllowCredentials {
		return nil, ErrCredentialedWildcard
	}

	methods := make(map[string]bool, len(p.AllowedMethods))
	for _, m := range p.AllowedMethods {
		methods[strings.ToUpper(m)] = true
	}
	allowMethods := strings.Join(p.AllowedMethods, ", ")
	allowHeaders := strings.Join(p.AllowedHeaders, ", ")
	exposeHeaders := strings.Join(p.ExposedHeaders, ", ")
	maxAge := strconv.Itoa(int(p.MaxAge.Seconds()))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			// Unconditional, even when no Origin arrived: a cache that stored
			// this response must know the answer depends on the origin.
			h.Add("Vary", "Origin")

			origin := r.Header.Get("Origin")
			// A preflight is an OPTIONS request that names the method it is
			// asking about. OPTIONS without that header is an ordinary
			// request for the handler.
			isPreflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
			if isPreflight {
				h.Add("Vary", "Access-Control-Request-Method")
				h.Add("Vary", "Access-Control-Request-Headers")
			}

			allowed := origin != "" && (wildcard || origins[origin])
			if allowed {
				if wildcard {
					h.Set("Access-Control-Allow-Origin", "*")
				} else {
					// Echo the exact origin, never a computed prefix or
					// suffix match: "https://app.example.com.evil.test" ends
					// with your domain and is not your domain.
					h.Set("Access-Control-Allow-Origin", origin)
				}
				if p.AllowCredentials {
					h.Set("Access-Control-Allow-Credentials", "true")
				}
			}

			if !isPreflight {
				if allowed && exposeHeaders != "" {
					// Permission to read the body is not permission to read
					// the headers: without this, only the seven safelisted
					// response headers reach the page, and the Retry-After on
					// a 429 is not one of them.
					h.Set("Access-Control-Expose-Headers", exposeHeaders)
				}
				// Note what does not happen here: an unlisted origin is not
				// refused. CORS tells the browser what a page may read; it is
				// not authentication, and a non-browser client ignores it.
				next.ServeHTTP(w, r)
				return
			}

			if allowed && methods[strings.ToUpper(r.Header.Get("Access-Control-Request-Method"))] {
				h.Set("Access-Control-Allow-Methods", allowMethods)
				if allowHeaders != "" {
					h.Set("Access-Control-Allow-Headers", allowHeaders)
				}
				if p.MaxAge > 0 {
					h.Set("Access-Control-Max-Age", maxAge)
				}
			}
			// Answered either way, with or without permission headers:
			// withholding them is how a preflight is denied.
			w.WriteHeader(http.StatusNoContent)
		})
	}, nil
}
