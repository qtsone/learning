package scalability

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// SessionStore is where session state lives *outside* any single instance —
// the stand-in for Redis or a database shared by the whole fleet.
type SessionStore interface {
	Save(token, user string)
	Load(token string) (user string, ok bool)
}

// MemoryStore is a concurrency-safe SessionStore for tests.
type MemoryStore struct {
	mu sync.Mutex
	m  map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{m: make(map[string]string)}
}

func (s *MemoryStore) Save(token, user string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[token] = user
}

func (s *MemoryStore) Load(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.m[token]
	return u, ok
}

// App is one instance of the service behind a load balancer. It holds no
// session state of its own, so any instance can serve any request.
type App struct {
	store SessionStore
}

func NewApp(store SessionStore) *App {
	return &App{store: store}
}

// Login creates a session for user and returns its token.
func (a *App) Login(user string) string {
	token := newToken()
	a.store.Save(token, user)
	return token
}

// Whoami resolves a session token to its user; ok is false for unknown tokens.
func (a *App) Whoami(token string) (string, bool) {
	return a.store.Load(token)
}

func newToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
