package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// SessionCookieName is the cookie the browser carries. In production you would
// name it "__Host-session": browsers enforce that a "__Host-" cookie is
// Secure, Path=/ and has no Domain, so a sibling subdomain cannot set one.
const SessionCookieName = "session"

// dummyPassword exists only so an unknown username still costs one bcrypt
// verification. It is not a secret; its job is to make failed logins take the
// same work whether the account exists or not.
const dummyPassword = "there-is-no-account-with-this-password"

// Config wires the Service. Every zero value is a sane default except
// CookieSecure, which defaults to insecure so plain-HTTP local development
// works — set it true everywhere else.
type Config struct {
	Clock        Clock
	Hasher       PasswordHasher
	SessionTTL   time.Duration
	CookieSecure bool
}

type Service struct {
	users        *userStore
	sessions     *SessionStore
	hasher       PasswordHasher
	ttl          time.Duration
	cookieSecure bool
	dummyHash    string
}

func NewService(cfg Config) *Service {
	clock := cfg.Clock
	if clock == nil {
		clock = RealClock{}
	}
	hasher := cfg.Hasher
	if hasher == nil {
		hasher = Hasher{}
	}
	ttl := cfg.SessionTTL
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	// If this fails the dummy hash is empty, Verify still fails, and the login
	// path still answers 401 — no error path worth surfacing at construction.
	dummyHash, _ := hasher.Hash(dummyPassword)
	return &Service{
		users:        newUserStore(),
		sessions:     NewSessionStore(clock, ttl),
		hasher:       hasher,
		ttl:          ttl,
		cookieSecure: cfg.CookieSecure,
		dummyHash:    dummyHash,
	}
}

// Sessions exposes the store for wiring and tests.
func (s *Service) Sessions() *SessionStore { return s.sessions }

// Routes is the service's HTTP surface. Note where Authenticate sits: routes
// that need an identity are wrapped, routes that mint one cannot be.
func (s *Service) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", s.HandleLogin)
	mux.HandleFunc("POST /logout", s.HandleLogout)
	mux.Handle("GET /me", s.Authenticate(http.HandlerFunc(handleMe)))
	mux.Handle("POST /me/promote", s.Authenticate(http.HandlerFunc(s.HandlePromote)))
	return mux
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		// Unreachable behind Authenticate; a 500 here means the wiring broke.
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeData(w, http.StatusOK, user)
}

// Register creates an account. There is no HTTP handler for it on purpose:
// real signup flows differ wildly (invite codes, e-mail confirmation, admin
// provisioning), so the package offers the operation and your service decides
// how it is reached.
func (s *Service) Register(username, password string) (User, error) {
	username = strings.TrimSpace(username)
	if len(username) < 3 {
		return User{}, ErrInvalidUsername
	}
	if len(password) < MinPasswordBytes {
		return User{}, ErrPasswordTooShort
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return User{}, err
	}
	return s.users.create(username, hash)
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// HandleLogin verifies credentials and issues a fresh session.
func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var creds credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	user, found := s.users.findByUsername(strings.TrimSpace(creds.Username))
	// Always verify something: skipping the hash for unknown users turns login
	// latency into a "does this account exist?" oracle.
	hash := s.dummyHash
	if found {
		hash = user.PasswordHash
	}
	err := s.hasher.Verify(hash, creds.Password)
	if err != nil && !errors.Is(err, ErrBadCredentials) {
		slog.Error("verify password", "err", err)
	}
	if err != nil || !found {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Session fixation defence: whatever id the client arrived with, it does
	// not become the id of the authenticated session.
	if c, cerr := r.Cookie(SessionCookieName); cerr == nil {
		s.sessions.Delete(c.Value)
	}
	sess, err := s.sessions.New(user.ID)
	if err != nil {
		slog.Error("create session", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.setSessionCookie(w, sess)
	writeData(w, http.StatusOK, user)
}

// HandleLogout destroys the server-side session and clears the cookie. Both
// halves matter: clearing only the cookie leaves a session a stolen copy of
// the id can still use.
func (s *Service) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(SessionCookieName); err == nil {
		s.sessions.Delete(c.Value)
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// HandlePromote grants the caller RoleAdmin and rotates the session id, so an
// id captured while the account was ordinary is not an admin id now.
func (s *Service) HandlePromote(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	current, hasSession := SessionFromContext(r.Context())
	if !ok || !hasSession {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	updated, ok := s.users.setRole(user.ID, RoleAdmin)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	sess, err := s.sessions.Rotate(current.ID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	s.setSessionCookie(w, sess)
	writeData(w, http.StatusOK, updated)
}

// Authenticate is the single place that turns a cookie into an identity.
func (s *Service) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(SessionCookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		sess, ok := s.sessions.Lookup(c.Value)
		if !ok {
			s.clearSessionCookie(w)
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		user, ok := s.users.findByID(sess.UserID)
		if !ok {
			// The account vanished under a live session: drop the session too.
			s.sessions.Delete(sess.ID)
			s.clearSessionCookie(w)
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r.WithContext(withAuth(r.Context(), user, sess)))
	})
}

func (s *Service) setSessionCookie(w http.ResponseWriter, sess Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		MaxAge:   int(s.ttl.Seconds()),
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Service) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
