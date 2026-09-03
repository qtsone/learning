package auth

import (
	"net/http"
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

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register creates an account with RoleUser. There is no HTTP handler for it
// on purpose: real signup flows differ wildly (invite codes, e-mail
// confirmation, admin provisioning), so the package offers the operation and
// your service decides how it is reached.
//
// Trim the username; reject one shorter than 3 characters with
// ErrInvalidUsername and a password shorter than MinPasswordBytes with
// ErrPasswordTooShort. Store the hash, never the password.
func (s *Service) Register(username, password string) (User, error) {
	// TODO: validate, hash, hand to s.users.create.
	return User{}, nil
}

// HandleLogin decodes {"username","password"}, verifies the password, and
// issues a fresh session.
//
//   - a body that is not JSON → 400 "invalid JSON"
//   - unknown user or wrong password → 401 "invalid credentials", and both
//     must cost one Verify call: for an unknown user, verify the submitted
//     password against s.dummyHash and throw the result away
//   - success → a session cookie (setSessionCookie) and 200 with the user
//     under "data"
//
// Before issuing the session, delete whatever session id the request already
// carried. A login must never keep a client-supplied id.
func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// TODO: implement.
}

// HandleLogout destroys the server-side session named by the cookie, clears
// the cookie, and answers 204 — whether or not there was a session. Clearing
// only the cookie leaves a session a stolen copy of the id can still use.
func (s *Service) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// TODO: implement.
}

// HandlePromote grants the caller RoleAdmin and rotates the session id, so an
// id captured while the account was ordinary is not an admin id now. It runs
// behind Authenticate, so the identity and the session are already in the
// request context. Answer 200 with the updated user under "data".
func (s *Service) HandlePromote(w http.ResponseWriter, r *http.Request) {
	// TODO: implement.
}

// Authenticate is the single place that turns a cookie into an identity: no
// cookie, no live session, or no such user → 401 "authentication required",
// and a cookie naming a dead session is cleared on the way out. Otherwise put
// the user and session in the request context with withAuth and call next.
func (s *Service) Authenticate(next http.Handler) http.Handler {
	// TODO: replace this pass-through with the real middleware.
	return next
}

// setSessionCookie writes the session cookie: Path "/", MaxAge equal to the
// store's TTL in seconds, HttpOnly, SameSite=Lax, and Secure when configured.
func (s *Service) setSessionCookie(w http.ResponseWriter, sess Session) {
	// TODO: implement.
}

// clearSessionCookie overwrites the cookie with an empty value and MaxAge -1
// (delete now), keeping every other attribute identical — a browser only
// replaces a cookie when name, path and domain all match.
func (s *Service) clearSessionCookie(w http.ResponseWriter) {
	// TODO: implement.
}
