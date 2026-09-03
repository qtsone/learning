package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordBytes is policy: long passphrases beat short "complex" ones.
	MinPasswordBytes = 12
	// MaxPasswordBytes is mechanics: the bcrypt algorithm ignores everything
	// past 72 bytes, and many implementations truncate silently — Go's
	// x/crypto refuses outright with bcrypt.ErrPasswordTooLong. We reject at
	// our own boundary, with our own sentinel, so the behaviour is ours
	// rather than the library's.
	MaxPasswordBytes = 72
)

var (
	// ErrBadCredentials is deliberately one error for "no such user" and
	// "wrong password": callers must not be able to tell them apart.
	ErrBadCredentials = errors.New("auth: invalid credentials")

	ErrPasswordTooShort = errors.New("auth: password too short")
	ErrPasswordTooLong  = errors.New("auth: password too long")
)

// PasswordHasher is the seam the Service depends on, so a test can wrap the
// real hasher and observe how often it is asked to verify.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(encoded, password string) error
}

// Hasher implements PasswordHasher with bcrypt.
type Hasher struct {
	// Cost is bcrypt's work factor, doubling per unit. Zero means
	// bcrypt.DefaultCost. Tests use bcrypt.MinCost to stay fast.
	Cost int
}

func (h Hasher) cost() int {
	if h.Cost == 0 {
		return bcrypt.DefaultCost
	}
	return h.Cost
}

// Hash returns the encoded bcrypt hash of password — salt and cost included in
// the string, so nothing else has to be stored.
//
// A password longer than MaxPasswordBytes must fail with ErrPasswordTooLong —
// our own sentinel, returned before bcrypt gets the chance to reject it with
// its own.
func (h Hasher) Hash(password string) (string, error) {
	// TODO: reject overlong passwords, then hash with bcrypt at h.cost().
	// Wrap any other bcrypt error with %w; never return the password.
	return "", nil
}

// Verify checks password against an encoded hash: nil when it matches,
// ErrBadCredentials when it does not.
//
// Any other failure (a truncated or corrupt stored hash) is a *server* fact,
// not a user fact: return it wrapped so it can be logged and distinguished.
func (h Hasher) Verify(encoded, password string) error {
	// TODO: compare with bcrypt and translate the three outcomes.
	return nil
}

// NeedsRehash reports whether encoded was produced below the current cost —
// the signal to re-hash a password during a successful login, the one moment
// the plaintext is in your hands. An unreadable hash also needs replacing.
func (h Hasher) NeedsRehash(encoded string) bool {
	// TODO: read the cost out of the encoded hash and compare with h.cost().
	return false
}
