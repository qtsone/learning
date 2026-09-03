package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// SessionIDBytes is 256 bits of entropy: guessing is off the table, so the
	// store can look ids up in a map instead of comparing them byte by byte.
	SessionIDBytes = 32

	// DefaultSessionTTL is an absolute lifetime, not an idle timeout.
	DefaultSessionTTL = 12 * time.Hour
)

// ErrNoSession means the id names nothing live: unknown, already rotated,
// logged out, or expired. One error, because the caller's answer is the same.
var ErrNoSession = errors.New("auth: no such session")

// Session is the server-side record. The id in the cookie is a lookup key with
// no meaning of its own — that is the whole point of server-side sessions.
type Session struct {
	ID        string
	UserID    string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

func (s Session) expired(now time.Time) bool { return !now.Before(s.ExpiresAt) }

// NewSessionID returns a fresh id from crypto/rand. math/rand/v2 is a fine
// generator and the wrong tool here: the package makes no unpredictability
// guarantee, and its explicit sources (PCG, a seeded ChaCha8) are reproducible
// from their seed. Only crypto/rand promises what a session id needs.
func NewSessionID() (string, error) {
	b := make([]byte, SessionIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SessionStore keeps live sessions in memory. Swap the map for a table or a
// Redis client and the interesting property survives: the server can delete a
// session, which is exactly what a signed token cannot offer.
type SessionStore struct {
	clock Clock
	ttl   time.Duration

	mu   sync.Mutex
	byID map[string]Session
}

func NewSessionStore(clock Clock, ttl time.Duration) *SessionStore {
	if clock == nil {
		clock = RealClock{}
	}
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	return &SessionStore{clock: clock, ttl: ttl, byID: make(map[string]Session)}
}

// New issues a session for userID, expiring ttl from now.
func (s *SessionStore) New(userID string) (Session, error) {
	id, err := NewSessionID()
	if err != nil {
		return Session{}, err
	}
	now := s.clock.Now()
	sess := Session{ID: id, UserID: userID, IssuedAt: now, ExpiresAt: now.Add(s.ttl)}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[id] = sess
	return sess, nil
}

// Lookup returns the live session for id. An expired session is dropped on the
// way out: expiry is enforced on read, never trusted from the cookie.
func (s *SessionStore) Lookup(id string) (Session, bool) {
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.byID[id]
	if !ok {
		return Session{}, false
	}
	if sess.expired(now) {
		delete(s.byID, id)
		return Session{}, false
	}
	return sess, true
}

// Rotate replaces a live session with a new id for the same user and returns
// it. The old id stops working immediately — that is the anti-fixation and
// privilege-change property the whole exercise turns on.
func (s *SessionStore) Rotate(id string) (Session, error) {
	newID, err := NewSessionID()
	if err != nil {
		return Session{}, err
	}
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.byID[id]
	if !ok || old.expired(now) {
		delete(s.byID, id)
		return Session{}, ErrNoSession
	}
	delete(s.byID, id)
	sess := Session{ID: newID, UserID: old.UserID, IssuedAt: now, ExpiresAt: now.Add(s.ttl)}
	s.byID[newID] = sess
	return sess, nil
}

// Delete drops a session. Unknown ids are not an error: logout is idempotent.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byID, id)
}

// Len reports how many sessions are stored, expired ones included.
func (s *SessionStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.byID)
}
