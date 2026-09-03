package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// Middleware wraps a handler with behavior that runs around it.
type Middleware func(http.Handler) http.Handler

const requestIDHeader = "X-Request-ID"

// timeoutBody is what a client gets when Timeout fires. It is the same
// envelope shape as every other error, so a client only has to know one.
const timeoutBody = `{"error":{"message":"request timeout"}}`

type (
	requestIDKey struct{}
	clientKey    struct{}
)

// WithRequestID returns a copy of ctx carrying id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFrom returns the id RequestID put on ctx, or "".
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// WithClient returns a copy of ctx carrying the authenticated client's name.
func WithClient(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, clientKey{}, name)
}

// ClientFrom returns the client name Auth put on ctx, or "" when the request
// was never authenticated.
func ClientFrom(ctx context.Context) string {
	name, _ := ctx.Value(clientKey{}).(string)
	return name
}

// Chain applies middleware so the FIRST one listed is outermost:
// Chain(h, A, B) serves requests as A(B(h)).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// RequestID reuses an incoming X-Request-ID header or generates one, puts it
// on the request context, and echoes it on the response so a caller can
// quote it in a bug report.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = strconv.FormatUint(rand.Uint64(), 16)
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(WithRequestID(r.Context(), id)))
	})
}

// statusWriter records the status a handler wrote so middleware can read it
// afterwards. The zero value of status is set by callers to 200, because a
// handler that never calls WriteHeader still produced a 200.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.NewResponseController reach the real ResponseWriter, so
// handlers keep access to flushing and hijacking through this wrapper.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// AccessLog returns a middleware that emits exactly one structured line per
// request through logger. The attribute names are a contract: dashboards and
// alerts are written against them, so they are not free to drift.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(sw, r)
			logger.LogAttrs(r.Context(), slog.LevelInfo, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
				slog.String("request_id", RequestIDFrom(r.Context())),
			)
			// Note what is NOT here: the Authorization header, the request
			// body, the query string. Logs get copied, shipped, and kept.
		})
	}
}

// Recover returns a middleware that turns a handler panic into a 500
// response instead of a killed connection.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				p := recover()
				if p == nil {
					return
				}
				logger.LogAttrs(r.Context(), slog.LevelError, "handler panic",
					slog.Any("panic", p),
					slog.String("path", r.URL.Path),
					slog.String("request_id", RequestIDFrom(r.Context())),
					slog.String("stack", string(debug.Stack())),
				)
				writeError(w, http.StatusInternalServerError, "internal error", nil)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout returns a middleware that bounds how long a handler may take.
// http.TimeoutHandler runs the handler on its own goroutine with a deadline
// on the request context and buffers its writes; when the deadline passes
// first, the client gets a 503 and the handler's later writes are discarded.
// The handler goroutine itself is NOT killed — nothing in Go can do that —
// which is why every layer below takes a context and checks it.
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, timeoutBody)
	}
}

// Auth returns a middleware that accepts only requests carrying a bearer
// token from tokens, a map of client name to secret token. The authenticated
// client's name goes on the request context.
//
// Threat model, stated plainly: these are shared secrets, so the transport
// must be TLS (a bearer token on plaintext HTTP is a token in every proxy
// log between here and the client), they must arrive in a header and never
// in a URL, and they are compared as SHA-256 digests in constant time so
// neither the token's value nor its length can be recovered from response
// timing. Rotating one means editing the environment and restarting.
func Auth(tokens map[string]string) Middleware {
	type client struct {
		name   string
		digest [sha256.Size]byte
	}
	// Digests are computed once at construction: hashing per request would
	// add work to the hot path for no security gain.
	clients := make([]client, 0, len(tokens))
	for name, secret := range tokens {
		clients = append(clients, client{name: name, digest: sha256.Sum256([]byte(secret))})
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if presented, ok := bearerToken(r.Header.Get("Authorization")); ok {
				digest := sha256.Sum256([]byte(presented))
				for _, c := range clients {
					if subtle.ConstantTimeCompare(digest[:], c.digest[:]) == 1 {
						next.ServeHTTP(w, r.WithContext(WithClient(r.Context(), c.name)))
						return
					}
				}
			}
			// One answer for "no token", "wrong token" and "unknown
			// client": a 401 that tells an attacker nothing it did not
			// already know.
			w.Header().Set("WWW-Authenticate", `Bearer realm="taskd"`)
			writeError(w, http.StatusUnauthorized, "unauthorized", nil)
		})
	}
}

// bearerToken extracts the credential from an Authorization header. RFC 7235
// makes the scheme case-insensitive, so "bearer" is as valid as "Bearer".
func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}
