package board

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// SessionCookieName is the cookie the browser carries. In production name it
// "__Host-session": browsers enforce that a "__Host-" cookie is Secure, Path=/
// and domain-locked, so a compromised sibling subdomain cannot plant one.
const SessionCookieName = "session"

// SessionIDBytes is how much entropy a session id carries. It comes from
// crypto/rand: math/rand/v2 is a fine generator and the wrong tool here,
// because the package makes no unpredictability guarantee.
const SessionIDBytes = 32

// DefaultSessionTTL is the absolute lifetime of a session. Expiry is enforced
// server-side on every read; a cookie's Max-Age is advice to a client you do
// not control.
const DefaultSessionTTL = 12 * time.Hour

// PasswordHasher is the password verifier the service depends on. It is an
// interface so the test suite can substitute a cheap one — bcrypt at production
// cost is deliberately slow, which is the point of it.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) error
}

// BcryptHasher stores passwords with bcrypt: a per-password salt and a tunable
// cost, encoded in one self-describing string. A general-purpose hash like
// SHA-256 is engineered to be fast, which is exactly the property an offline
// cracker wants.
type BcryptHasher struct {
	// Cost is the work factor. Pick the highest your login endpoint can
	// afford — roughly 100-250 ms per hash on production hardware.
	Cost int
}

// Hash returns a salted bcrypt hash of password.
func (h BcryptHasher) Hash(password string) (string, error) {
	cost := h.Cost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	// The bcrypt algorithm ignores everything past 72 bytes, and many
	// implementations truncate silently — Go's x/crypto refuses outright with
	// bcrypt.ErrPasswordTooLong. Refuse at our own boundary, with our own
	// error, so the behaviour is ours rather than the library's.
	if len(password) > 72 {
		return "", errors.New("board: password too long")
	}
	sum, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(sum), nil
}

// Verify reports whether password matches hash, in constant time.
func (h BcryptHasher) Verify(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrBadCredentials
	}
	return nil
}

// Session is one server-side login. The id in the cookie means nothing by
// itself; it is a lookup key, and deleting the row here ends the session
// everywhere — which is the property tokens cannot offer.
type Session struct {
	ID        string
	UserID    string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// SessionStore holds live sessions in memory. One process, so a restart logs
// everyone out and two instances would disagree; the fix is the same three
// methods over a shared store, and nothing above this type changes.
type SessionStore struct {
	clock Clock
	ttl   time.Duration

	mu       sync.Mutex
	sessions map[string]Session
}

// NewSessionStore builds a store whose sessions live for ttl.
func NewSessionStore(clock Clock, ttl time.Duration) *SessionStore {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	return &SessionStore{clock: clock, ttl: ttl, sessions: make(map[string]Session)}
}

// TTL is the absolute session lifetime, which is also the cookie's Max-Age.
func (s *SessionStore) TTL() time.Duration { return s.ttl }

// New mints a session id from crypto/rand and stores the session.
func (s *SessionStore) New(userID string) (Session, error) {
	buf := make([]byte, SessionIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return Session{}, fmt.Errorf("session id: %w", err)
	}
	now := s.clock.Now()
	sess := Session{
		ID:        base64.RawURLEncoding.EncodeToString(buf),
		UserID:    userID,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.ttl),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
	return sess, nil
}

// Get returns a live session. Expiry is enforced here, on read, against the
// injected clock, and an expired session is dropped on the way out.
func (s *SessionStore) Get(id string) (Session, bool) {
	if id == "" {
		return Session{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return Session{}, false
	}
	if !s.clock.Now().Before(sess.ExpiresAt) {
		delete(s.sessions, id)
		return Session{}, false
	}
	return sess, true
}

// Delete ends one session.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// DeleteUser ends every session belonging to a user and reports how many it
// destroyed. This is the revocation primitive: what you reach for when the
// answer to "what may this session do" has changed underneath it.
func (s *SessionStore) DeleteUser(userID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for id, sess := range s.sessions {
		if sess.UserID == userID {
			delete(s.sessions, id)
			n++
		}
	}
	return n
}

// Cleanup deletes every session past its expiry and reports how many it
// removed. Get enforces expiry on read, which keeps an expired session from
// *working* but never removes it: a caller who logs in and closes the tab
// leaves a row nobody reads again. Without a sweep the map only grows, and it
// grows once per login attempt that succeeds.
func (s *SessionStore) Cleanup() int {
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for id, sess := range s.sessions {
		if !now.Before(sess.ExpiresAt) {
			delete(s.sessions, id)
			n++
		}
	}
	return n
}

// Count reports how many sessions are stored, live or not.
func (s *SessionStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

// Subject is who a request is from, as far as the policy is concerned. It is
// built once per request, from the session, and never from anything in the
// body or the query string.
type Subject struct {
	UserID    string
	Role      Role
	SessionID string
}

// Anonymous reports whether the request carried no usable identity.
func (s Subject) Anonymous() bool { return s.UserID == "" }

type subjectCtxKey struct{}

// WithSubject returns a context carrying sub. Unexported key, exported
// accessor: nothing outside this package can fabricate an identity into a context.
func WithSubject(ctx context.Context, sub Subject) context.Context {
	return context.WithValue(ctx, subjectCtxKey{}, sub)
}

// SubjectFrom reads the identity attached by the authentication middleware. The
// zero Subject — anonymous — is the honest answer when there is none.
func SubjectFrom(ctx context.Context) Subject {
	sub, _ := ctx.Value(subjectCtxKey{}).(Subject)
	return sub
}
