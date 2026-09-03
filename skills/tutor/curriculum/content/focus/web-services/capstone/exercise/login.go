package board

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// Register creates an account, storing only the hash. There is no HTTP handler
// for it on purpose: real signup flows differ wildly (invite codes, e-mail
// confirmation, admin provisioning), so the package offers the operation and a
// deployment decides how it is reached.
func (s *Server) Register(ctx context.Context, username, password string, role Role) (User, error) {
	username = strings.TrimSpace(username)
	if len(username) < 3 {
		return User{}, errors.New("board: username must be at least 3 characters")
	}
	if len(password) < 8 {
		return User{}, errors.New("board: password must be at least 8 characters")
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return User{}, err
	}
	u := User{ID: "u-" + username, Username: username, Role: role, PasswordHash: hash}
	if err := s.store.CreateUser(ctx, u); err != nil {
		return User{}, err
	}
	return u, nil
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// HandleLogin verifies a password and issues a fresh session.
//
// Two properties are worth naming. An unknown username and a wrong password
// produce the same status, the same body and the same *work* — the dummy hash
// is verified when there is no user — so login latency is not an answer to
// "does this account exist?". And whatever session id the request arrived with
// is destroyed before a new one is minted: an id the client chose must never
// become the id of an authenticated session (session fixation).
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var creds credentials
	if rerr := DecodeJSON(w, r, DefaultMaxBodyBytes, &creds); rerr != nil {
		writeRequestError(w, rerr)
		return
	}

	user, err := s.store.UserByUsername(r.Context(), creds.Username)
	hash := s.dummyHash
	if err == nil {
		hash = user.PasswordHash
	} else if !errors.Is(err, ErrNotFound) {
		s.log.Error("login lookup failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if verr := s.hasher.Verify(hash, creds.Password); verr != nil || err != nil {
		// Deliberately no username: people type their password into that
		// field often enough that logging it turns the log into a
		// plaintext credential store.
		s.log.Warn("login failed")
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if c, cerr := r.Cookie(SessionCookieName); cerr == nil {
		s.sessions.Delete(c.Value)
	}
	sess, err := s.sessions.New(user.ID)
	if err != nil {
		s.log.Error("session mint failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.setSessionCookie(w, sess)
	writeData(w, http.StatusOK, user)
}

// HandleLogout destroys the server-side session and clears the cookie, and
// answers 204 whether or not there was a session. Clearing only the cookie
// would leave a session a stolen copy of the id can still use.
func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(SessionCookieName); err == nil {
		s.sessions.Delete(c.Value)
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// setSessionCookie writes the session cookie. Each attribute is a defence:
// HttpOnly keeps an XSS bug from reading it, Secure keeps it off plaintext,
// SameSite=Lax keeps it off cross-site POSTs, and Max-Age tells the browser
// when to forget it — while the server decides for real.
func (s *Server) setSessionCookie(w http.ResponseWriter, sess Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		MaxAge:   int(s.sessions.TTL().Seconds()),
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie overwrites the cookie with MaxAge -1, keeping every other
// attribute identical — a browser only replaces a cookie when name, path and
// domain all match.
func (s *Server) clearSessionCookie(w http.ResponseWriter) {
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
