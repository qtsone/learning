package auth

import (
	"errors"
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

// NewSessionID returns a fresh session id: SessionIDBytes of randomness,
// base64url-encoded without padding.
//
// Which randomness matters more than the encoding. math/rand/v2 is a fine
// generator and a catastrophic choice here.
func NewSessionID() (string, error) {
	// TODO: read SessionIDBytes from crypto/rand and encode them.
	return "", nil
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

// New issues a session for userID with IssuedAt of now and ExpiresAt of
// now plus the store's TTL, where now comes from the injected Clock.
func (s *SessionStore) New(userID string) (Session, error) {
	// TODO: mint an id, build the Session, store it under the id.
	return Session{}, nil
}

// Lookup returns the live session for id. An expired session must be reported
// as absent *and* dropped from the map: expiry is enforced here on every read,
// never trusted from the cookie.
func (s *SessionStore) Lookup(id string) (Session, bool) {
	// TODO: look the id up, enforce expiry against the Clock.
	return Session{}, false
}

// Rotate replaces a live session with a new id for the same user and returns
// it, with fresh IssuedAt/ExpiresAt. The old id must stop working immediately.
// An unknown or expired id is ErrNoSession.
func (s *SessionStore) Rotate(id string) (Session, error) {
	// TODO: implement. Mind the mutex: mint the id before taking the lock.
	return Session{}, nil
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
