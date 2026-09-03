package auth

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Roles this service knows about. The authorization lesson later in this pack
// turns roles into a real policy; here a role is just a field that changes.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

var (
	// ErrUserExists means the username is taken. Never surface it from a
	// public registration endpoint verbatim: "that name is taken" is a user
	// enumeration oracle. Registration flows answer it with e-mail, not HTTP.
	ErrUserExists = errors.New("auth: username already taken")

	ErrInvalidUsername = errors.New("auth: invalid username")
)

// User is the account record. Everything in it is safe to send to the account
// owner except the hash, which the json:"-" tag keeps out of every response
// that ever marshals a User — including ones you write months from now.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
}

// userStore is an in-memory account table. A real service puts this in
// Postgres or SQLite behind the same three methods; the mutex here stands in
// for the database's own concurrency control.
type userStore struct {
	mu       sync.Mutex
	byID     map[string]User
	idByName map[string]string // lower-cased username → id
	next     int
}

func newUserStore() *userStore {
	return &userStore{byID: make(map[string]User), idByName: make(map[string]string)}
}

// create stores a new user with role RoleUser. Usernames are compared
// case-insensitively so "Ada" and "ada" cannot both be registered.
func (s *userStore) create(username, hash string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.ToLower(username)
	if _, taken := s.idByName[key]; taken {
		return User{}, ErrUserExists
	}
	s.next++
	u := User{
		ID:           fmt.Sprintf("u%d", s.next),
		Username:     username,
		Role:         RoleUser,
		PasswordHash: hash,
	}
	s.byID[u.ID] = u
	s.idByName[key] = u.ID
	return u, nil
}

func (s *userStore) findByUsername(username string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.idByName[strings.ToLower(username)]
	if !ok {
		return User{}, false
	}
	u, ok := s.byID[id]
	return u, ok
}

func (s *userStore) findByID(id string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byID[id]
	return u, ok
}

func (s *userStore) setRole(id, role string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byID[id]
	if !ok {
		return User{}, false
	}
	u.Role = role
	s.byID[id] = u
	return u, true
}
