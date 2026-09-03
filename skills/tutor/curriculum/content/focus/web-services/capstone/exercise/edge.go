package board

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// DefaultMaxBodyBytes is the ceiling for a JSON request body. Without it,
// `json.NewDecoder(r.Body).Decode(&v)` decodes until the value ends or the
// reader does — and the client decides which comes first.
const DefaultMaxBodyBytes int64 = 32 << 10

// unknownFieldPrefix is how encoding/json reports a rejected field. There is no
// typed error for it, so the string is load-bearing; keeping it in one constant
// means one place to fix if it ever changes.
const unknownFieldPrefix = `json: unknown field `

// Middleware is one link in the chain. Order is part of the design, so the
// chain is written down in one place rather than assembled per route.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares so that the first one listed is the outermost.
func Chain(h http.Handler, ms ...Middleware) http.Handler {
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i](h)
	}
	return h
}

// Validator is implemented by request types that check their own fields.
// Returning every failure instead of the first one lets a client fix a whole
// form in one round trip.
type Validator interface {
	Validate() []FieldError
}

// DecodeJSON reads exactly one JSON value from r's body into dst and, when dst
// implements Validator, checks it. Every failure is a *RequestError carrying
// the status the caller should send: 415 for the wrong media type, 413 past the
// byte limit, 400 for "I could not parse this", 422 for "I parsed it and it
// does not mean anything".
func DecodeJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, dst any) *RequestError {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &RequestError{Status: http.StatusUnsupportedMediaType, Message: "Content-Type must be application/json"}
	}

	// MaxBytesReader, not io.LimitReader: LimitReader reports a clean EOF, so a
	// truncated body arrives as "malformed JSON" and the caller debugs JSON
	// that is perfectly valid.
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &RequestError{Status: http.StatusBadRequest, Message: "request body must contain a single JSON object"}
	}
	if v, ok := dst.(Validator); ok {
		if fields := v.Validate(); len(fields) > 0 {
			return &RequestError{Status: http.StatusUnprocessableEntity, Message: "validation failed", Fields: fields}
		}
	}
	return nil
}

func decodeError(err error) *RequestError {
	var (
		maxBytesErr *http.MaxBytesError
		syntaxErr   *json.SyntaxError
		typeErr     *json.UnmarshalTypeError
	)
	switch {
	case errors.As(err, &maxBytesErr):
		return &RequestError{
			Status:  http.StatusRequestEntityTooLarge,
			Message: fmt.Sprintf("request body must not exceed %d bytes", maxBytesErr.Limit),
		}
	case errors.As(err, &syntaxErr):
		return &RequestError{Status: http.StatusBadRequest, Message: fmt.Sprintf("malformed JSON at byte %d", syntaxErr.Offset)}
	case errors.Is(err, io.ErrUnexpectedEOF):
		return &RequestError{Status: http.StatusBadRequest, Message: "malformed JSON: body ended early"}
	case errors.As(err, &typeErr):
		if typeErr.Field == "" {
			return &RequestError{Status: http.StatusBadRequest, Message: "request body must be a JSON object"}
		}
		return &RequestError{
			Status:  http.StatusBadRequest,
			Message: "invalid request body",
			Fields:  []FieldError{{Field: typeErr.Field, Message: fmt.Sprintf("must be a %s", typeErr.Type)}},
		}
	case errors.Is(err, io.EOF):
		return &RequestError{Status: http.StatusBadRequest, Message: "request body must not be empty"}
	case strings.HasPrefix(err.Error(), unknownFieldPrefix):
		name, uerr := strconv.Unquote(strings.TrimPrefix(err.Error(), unknownFieldPrefix))
		if uerr != nil {
			return &RequestError{Status: http.StatusBadRequest, Message: "invalid request body"}
		}
		return &RequestError{
			Status:  http.StatusBadRequest,
			Message: "invalid request body",
			Fields:  []FieldError{{Field: name, Message: "unknown field"}},
		}
	default:
		// Never hand the raw decoder text to the caller: it can echo the body
		// back, and the body may be someone's data.
		return &RequestError{Status: http.StatusBadRequest, Message: "invalid request body"}
	}
}

// MaxTitleRunes bounds a task title. Runes, not bytes: the limit is a promise
// about what the user typed.
const MaxTitleRunes = 80

// CreateTaskRequest is the body of POST /tasks. There is no owner field, on
// purpose: ownership comes from the session, and a client that posts
// {"owner_id": "..."} must change nothing.
type CreateTaskRequest struct {
	Title string `json:"title"`
}

// Validate reports every broken rule in declaration order.
func (req CreateTaskRequest) Validate() []FieldError {
	var errs []FieldError
	switch {
	case strings.TrimSpace(req.Title) == "":
		errs = append(errs, FieldError{Field: "title", Message: "must not be empty"})
	case utf8.RuneCountInString(req.Title) > MaxTitleRunes:
		errs = append(errs, FieldError{Field: "title", Message: fmt.Sprintf("must be at most %d characters", MaxTitleRunes)})
	}
	return errs
}

// UpdateTaskRequest is the body of PATCH /tasks/{id}.
type UpdateTaskRequest struct {
	State TaskState `json:"state"`
}

// Validate reports whether the requested state exists.
func (req UpdateTaskRequest) Validate() []FieldError {
	if !ValidState(req.State) {
		return []FieldError{{Field: "state", Message: "must be one of open, doing, done"}}
	}
	return nil
}

// SetRoleRequest is the body of POST /users/{id}/role.
type SetRoleRequest struct {
	Role Role `json:"role"`
}

// Validate reports whether the requested role exists.
func (req SetRoleRequest) Validate() []FieldError {
	switch req.Role {
	case RoleMember, RoleAuditor, RoleAdmin:
		return nil
	}
	return []FieldError{{Field: "role", Message: "must be one of member, auditor, admin"}}
}

// bucket is one key's token bucket: the balance as of updated, with the tokens
// accrued since then computed on read — no goroutine per client.
type bucket struct {
	tokens  float64
	updated time.Time
}

// Limiter is a per-key token bucket, safe for concurrent use.
type Limiter struct {
	mu      sync.Mutex
	clock   Clock
	rate    float64
	burst   float64
	buckets map[string]*bucket
}

// NewLimiter grants ratePerSecond tokens per second per key, capped at burst.
func NewLimiter(ratePerSecond float64, burst int, clk Clock) *Limiter {
	if clk == nil {
		clk = RealClock{}
	}
	return &Limiter{clock: clk, rate: ratePerSecond, burst: float64(burst), buckets: make(map[string]*bucket)}
}

// Allow spends one token for key, reporting whether the request may proceed and
// how long until the next token exists.
func (l *Limiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, updated: now}
		l.buckets[key] = b
	} else if elapsed := now.Sub(b.updated); elapsed > 0 {
		b.tokens = math.Min(l.burst, b.tokens+elapsed.Seconds()*l.rate)
		b.updated = now
	}
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
}

// RefillWindow is how long a fully drained bucket takes to refill, and the
// shortest idle period Cleanup may safely use — which is why the janitor asks
// for it rather than picking a number.
func (l *Limiter) RefillWindow() time.Duration {
	if l.rate <= 0 {
		return 0
	}
	return time.Duration(l.burst / l.rate * float64(time.Second))
}

// Cleanup deletes every bucket untouched for at least idle. Without it the map
// grows once per distinct key seen, which is a memory-exhaustion vector handed
// to anyone with a range of addresses. Keep idle at least burst/rate: evicting
// a partly drained bucket hands its owner a full one.
func (l *Limiter) Cleanup(idle time.Duration) int {
	now := l.clock.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	removed := 0
	for key, b := range l.buckets {
		if now.Sub(b.updated) >= idle {
			delete(l.buckets, key)
			removed++
		}
	}
	return removed
}

// RetryAfterSeconds renders a Retry-After value: whole seconds, rounded *up*,
// never below 1. Rounding down invites the client straight into another refusal.
func RetryAfterSeconds(d time.Duration) string {
	seconds := int(math.Ceil(d.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

// ClientIP is the weakest useful rate-limit key: NAT puts thousands of people
// behind one address, and X-Forwarded-For is client-controlled text anyone can
// set, so it is read only when you know how many proxies you own. An
// authenticated account id is far better — which is why this service prefers
// one and falls back to here.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// SecurityHeaders sets the headers a JSON API earns its bytes with, on every
// response — including the ones produced by middleware that never reached a
// handler, which is why it belongs outermost.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// Stops the browser second-guessing the declared Content-Type: an
		// endpoint that reflects caller text and gets sniffed as HTML runs it.
		h.Set("X-Content-Type-Options", "nosniff")
		// If a response is ever rendered as a document it may load nothing, and
		// nobody may frame it. frame-ancestors replaces X-Frame-Options.
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		h.Set("Referrer-Policy", "no-referrer")
		if r.TLS != nil {
			// Only over TLS: on a plaintext response HSTS is ignored by spec,
			// and sending it anyway hides whether you are behind TLS at all.
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// ETag is the quoted hex SHA-256 of a response body — a strong validator, so a
// client that already holds these bytes can be answered with 304 and nothing
// else.
func ETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

// MatchETag implements If-None-Match: "*" matches anything, the header may be a
// comma-separated list, and comparison is weak (a W/ prefix is ignored).
func MatchETag(header, tag string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return false
	}
	if header == "*" {
		return true
	}
	want := strings.TrimPrefix(tag, "W/")
	for _, candidate := range strings.Split(header, ",") {
		if strings.TrimPrefix(strings.TrimSpace(candidate), "W/") == want {
			return true
		}
	}
	return false
}

// writeCachedJSON writes v under the success envelope with a strong ETag, and
// answers 304 when the caller already holds exactly these bytes. Headers set
// before the call survive onto the 304, which matters: a 304 that drops the
// caching headers teaches the client nothing about next time.
func writeCachedJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	body, err := json.Marshal(map[string]any{"data": v})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	tag := ETag(body)
	w.Header().Set("ETag", tag)
	if MatchETag(r.Header.Get("If-None-Match"), tag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		slog.Error("write response", "err", err)
	}
}

// StreamFrameTimeout bounds one *frame* of a streaming response. The server's
// WriteTimeout bounds a whole response, and an SSE response lasts hours, so the
// streaming handler pushes its own deadline forward frame by frame with
// http.ResponseController.SetWriteDeadline. Keep it comfortably above the
// heartbeat interval, or a healthy idle stream trips it.
const StreamFrameTimeout = 10 * time.Second

// NewHTTPServer wraps a handler in a server whose timeouts are decisions rather
// than defaults — every field of http.Server defaults to zero, and zero means
// no limit.
//
// WriteTimeout stays, unlike in most SSE examples, which drop it. It bounds
// *writing the response*, and an SSE stream is one response that lasts hours,
// so a global WriteTimeout would kill every stream on schedule — but the answer
// to that is to move the bound, not to remove it. GET /events resets its own
// deadline before every frame (StreamFrameTimeout), so a client that stops
// reading dies on the streaming route exactly as it does on every other one,
// and the hardening lesson's "all four timeouts" survives intact.
func NewHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,  // slow-header connections (Slowloris)
		ReadTimeout:       15 * time.Second, // a body dribbled one byte at a time
		WriteTimeout:      30 * time.Second, // a client that stops reading
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}
