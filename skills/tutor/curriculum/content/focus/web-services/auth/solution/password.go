package auth

import (
	"errors"
	"fmt"

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

// Hash returns the encoded bcrypt hash, salt and cost included.
func (h Hasher) Hash(password string) (string, error) {
	if len(password) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	encoded, err := bcrypt.GenerateFromPassword([]byte(password), h.cost())
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(encoded), nil
}

// Verify checks password against an encoded hash. A mismatch is
// ErrBadCredentials — a user fact. Anything else (a truncated or corrupt hash)
// is a server fact and comes back wrapped, for the log rather than the user.
func (h Hasher) Verify(encoded, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(encoded), []byte(password))
	switch {
	case err == nil:
		return nil
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return ErrBadCredentials
	default:
		return fmt.Errorf("compare password hash: %w", err)
	}
}

// NeedsRehash reports whether encoded was produced below the current cost. The
// only moment you hold a plaintext password is a successful login, so that is
// where the upgrade happens.
func (h Hasher) NeedsRehash(encoded string) bool {
	cost, err := bcrypt.Cost([]byte(encoded))
	return err != nil || cost < h.cost()
}
